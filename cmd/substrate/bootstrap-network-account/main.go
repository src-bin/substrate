package bootstrapnetworkaccount

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/availabilityzones"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
)

const (
	NATGatewaysFilename = "substrate.nat-gateways"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	ignoreServiceQuotas := flag.Bool("ignore-service-quotas", false, "ignore the appearance of any service quota being exhausted and continue anyway")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	cmdutil.MustChdir()
	flag.Usage = func() {
		ui.Print("Usage: substrate bootstrap-network-account [-auto-approve|-no-apply] [-ignore-service-quotas]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()

	cfg, err := cfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, "", time.Hour)
	if err != nil {
		ui.Fatal(err)
	}
	sess, err := awssessions.InSpecialAccount(accounts.Network, roles.NetworkAdministrator, awssessions.Config{
		FallbackToRootCredentials: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Gather the definitive list of environments and qualities first.
	environments, err := ui.EditFile(
		naming.EnvironmentsFilename,
		"the following environments are currently valid in your Substrate-managed infrastructure:",
		`list all your environments, one per line, in order of progression from e.g. development through e.g. production; your list MUST include "admin"`,
	)
	if err != nil {
		log.Fatal(err)
	}
	found := false
	for _, environment := range environments {
		if strings.ContainsAny(environment, " /") {
			ui.Fatal("environments cannot contain ' ' or '/'")
		}
		if environment == "peering" {
			ui.Fatal(`"peering" is a reserved environment name`)
		}
		found = found || environment == "admin"
	}
	if !found {
		ui.Fatal(`you must include "admin" in your list of environments`)
	}
	ui.Printf("using environments %s", strings.Join(environments, ", "))
	qualities, err := ui.EditFile(
		naming.QualitiesFilename,
		"the following qualities are currently valid in your Substrate-managed infrastructure:",
		`list all your qualities, one per line, in order from least to greatest quality (Substrate recommends starting out with just "default")`,
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(qualities) == 0 {
		ui.Fatal("you must name at least one quality")
	}
	for _, quality := range qualities {
		if strings.ContainsAny(quality, " /") {
			ui.Fatal("qualities cannot contain ' ' or '/'")
		}
	}
	ui.Printf("using qualities %s", strings.Join(qualities, ", "))

	// Combine all environments and qualities.  If a given combination doesn't
	// appear in substrate.valid-environment-quality-pairs.json then offer its
	// inclusion before validating the final document.
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	if len(veqpDoc.ValidEnvironmentQualityPairs) != 0 {
		ui.Print("you currently allow the following combinations of environment and quality in your Substrate-managed infrastructure:")
		for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
			ui.Printf("\t%-12s %s", eq.Environment, eq.Quality)
		}
	}
	if ui.Interactivity() == ui.FullyInteractive || ui.Interactivity() == ui.MinimallyInteractive && len(veqpDoc.ValidEnvironmentQualityPairs) == 0 {
		var ok bool
		if len(veqpDoc.ValidEnvironmentQualityPairs) != 0 {
			if ok, err = ui.Confirm("is this correct? (yes/no)"); err != nil {
				log.Fatal(err)
			}
		}
		if !ok {
			for _, environment := range environments {
				for _, quality := range qualities {
					if !veqpDoc.Valid(environment, quality) {
						ok, err := ui.Confirmf(`do you want to allow %s-quality infrastructure in your %s environment? (yes/no)`, quality, environment)
						if err != nil {
							log.Fatal(err)
						}
						if ok {
							veqpDoc.Ensure(environment, quality)
						}
					}
				}
			}
		}
	} else {
		ui.Print("if this is not correct, press ^C and re-run this command with -fully-interactive")
		time.Sleep(5e9) // give them a chance to ^C
	}
	if err := veqpDoc.Validate(environments, qualities); err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", veqpDoc)

	if _, err := regions.Select(); err != nil {
		log.Fatal(err)
	}

	natGateways, err := ui.ConfirmFile(
		NATGatewaysFilename,
		"do you want to provision NAT Gateways for IPv4 traffic from your private subnets to the Internet? (yes/no; costs about $100 per month per region per environment/quality pair)",
	)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%v", natGateways)

	// Configure the allocator for admin networks to use 192.168.0.0/16 and
	// 21-bit subnet masks which yields 2,048 IP addresses per VPC and 32
	// possible VPCs while keeping a tidy source IP address range for granting
	// SSH and other administrative access safely and easily.
	adminNetDoc, err := networks.ReadDocument(networks.AdminFilename, networks.RFC1918_192_168_0_0_16, 21)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", adminNetDoc)

	// Configure the allocator for normal (environment, quality) networks to use
	// 10.0.0.0/8 and 18-bit subnet masks which yields 16,384 IP addresses per
	// VPC and 1,024 possible VPCs.
	netDoc, err := networks.ReadDocument(networks.Filename, networks.RFC1918_10_0_0_0_8, 18)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", netDoc)

	// Write (or rewrite) Terraform resources that create the various
	// (environment, quality) networks.  Networks in the admin environment will
	// be created in the 192.168.0.0/16 CIDR block managed by adminNetDoc.
	ui.Printf("configuring networks for every environment and quality in %d regions", len(regions.Selected()))
	accountId := aws.StringValue(awssts.MustGetCallerIdentity(sts.New(sess)).Account)
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region := range regions.Selected() {
			ui.Spinf(
				"finding or assigning an IP address range to the %s %s network in %s",
				eq.Environment,
				eq.Quality,
				region,
			)
			var doc *networks.Document
			if eq.Environment == "admin" {
				doc = adminNetDoc
			} else {
				doc = netDoc
			}
			n, err := doc.Ensure(&networks.Network{
				Environment: eq.Environment,
				Quality:     eq.Quality,
				Region:      region,
			})
			if err != nil {
				log.Fatal(err)
			}
			//log.Printf("%+v", net)
			ui.Stop(n.IPv4)

			dirname := filepath.Join(terraform.RootModulesDirname, accounts.Network, eq.Environment, eq.Quality, region)

			file := terraform.NewFile()
			org := terraform.Organization{
				Label: terraform.Q("current"),
			}
			file.Push(org)
			tags := terraform.Tags{
				Environment: eq.Environment,
				Name:        fmt.Sprintf("%s-%s", eq.Environment, eq.Quality),
				Quality:     eq.Quality,
				Region:      region,
			}
			vpc := terraform.VPC{
				CidrBlock: terraform.Q(n.IPv4.String()),
				Label:     terraform.Label(tags),
				Tags:      tags,
			}
			file.Push(vpc)
			vpcAccoutrements(ctx, cfg, sess, natGateways, region, org, vpc, file)
			if err := file.Write(filepath.Join(dirname, "main.tf")); err != nil {
				log.Fatal(err)
			}

		}
	}

	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		log.Fatal(err)
	}
	cfg.SetCredentialsV1(ctx, creds.AccessKeyID, creds.SecretAccessKey, creds.SessionToken)
	cfg.Telemetry().FinalAccountId = accountId
	cfg.Telemetry().FinalRoleName = roles.NetworkAdministrator

	// Write to substrate.admin-networks.json and substrate.networks.json once
	// more so that, even if no changes were made, formatting changes and
	// SubstrateVersion are changed.
	if err := adminNetDoc.Write(); err != nil {
		log.Fatal(err)
	}
	if err := netDoc.Write(); err != nil {
		log.Fatal(err)
	}

	// Ensure the VPCs-per-region service quota and a few others aren't going to get in the way.
	var deadline time.Time
	if *ignoreServiceQuotas {
		deadline = time.Now()
	}
	ui.Print("raising the VPC, Internet, Egress-Only Internet, and NAT Gateway, and EIP service quotas in all your regions (this could take days, unfortunately; this program is safe to re-run)")
	adminNets := len(adminNetDoc.FindAll(&networks.Network{Region: regions.Selected()[0]})) // admin networks per region
	nets := len(netDoc.FindAll(&networks.Network{Region: regions.Selected()[0]}))           // (environment, quality) pairs per region
	for _, quota := range [][2]string{
		{"L-F678F1CE", "vpc"}, // VPCs per region
		{"L-45FE3B85", "vpc"}, // Egress-Only Internet Gateways per region
		{"L-A4707A72", "vpc"}, // Internet Gateways per region
	} {
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			sess,
			quota[0], quota[1],
			float64(adminNets+nets), // admin and non-admin VPCs per region, each with one of each type of Internet Gateway
			float64(adminNets+nets), // same value because they hassle us so much about raising the limit at all
			deadline,
		); err != nil {
			if _, ok := err.(awsservicequotas.DeadlinePassed); ok {
				ui.Print(err)
			} else {
				ui.Fatal(err)
			}
		}
	}
	if natGateways {
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			sess,
			"L-FE5A380F", "vpc", // NAT Gateways per availability zone
			float64(nets), // only non-admin networks get private subnets and thus NAT Gateways
			float64(nets), // same value because they hassle us so much about raising the limit at all
			deadline,
		); err != nil {
			if _, ok := err.(awsservicequotas.DeadlinePassed); ok {
				ui.Print(err)
			} else {
				ui.Fatal(err)
			}
		}
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			sess,
			"L-0263D0A3", "ec2", // EIPs per region
			float64(nets*availabilityzones.NumberPerNetwork), // NAT Gateways per AZ times the number of AZs per network
			float64(nets*availabilityzones.NumberPerNetwork), // same value because they hassle us so much about raising the limit at all
			deadline,
		); err != nil {
			if _, ok := err.(awsservicequotas.DeadlinePassed); ok {
				ui.Print(err)
			} else {
				ui.Fatal(err)
			}
		}
	}

	// Define networks for each environment and quality.  No peering yet as
	// it's difficult to reason about before all networks are created.
	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect multiple environments and qualities in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each enviornment and quality")
	}
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region := range regions.Selected() {
			dirname := filepath.Join(terraform.RootModulesDirname, accounts.Network, eq.Environment, eq.Quality, region)

			providersFile := terraform.NewFile()

			// The default provider for building out networks in this root module.
			providersFile.Push(terraform.ProviderFor(
				region,
				roles.Arn(accountId, roles.NetworkAdministrator),
			))

			// A provider for the substrate module to use, if for some reason it's
			// desired in this context.
			networkAccount, err := awsorgs.FindSpecialAccount(organizations.New(awssessions.Must(awssessions.InManagementAccount(
				roles.OrganizationReader,
				awssessions.Config{},
			))), accounts.Network)
			if err != nil {
				log.Fatal(err)
			}
			providersFile.Push(terraform.NetworkProviderFor(
				region,
				roles.Arn(aws.StringValue(networkAccount.Id), roles.Auditor),
			))
			if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
				log.Fatal(err)
			}

			if err := terraform.Root(dirname, region); err != nil {
				log.Fatal(err)
			}

			if err := terraform.Init(dirname); err != nil {
				log.Fatal(err)
			}

			if *noApply {
				err = terraform.Plan(dirname)
			} else {
				err = terraform.Apply(dirname, *autoApprove)
			}
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Now that all the networks exist, establish a fully-connected mesh of
	// peering connections within each environment's qualities and regions.
	peeringConnectionModule := terraform.PeeringConnectionModule()
	if err := peeringConnectionModule.Write(filepath.Join(terraform.ModulesDirname, "peering-connection")); err != nil {
		log.Fatal(err)
	}
	peeringConnections, err := networks.EnumeratePeeringConnections()
	if err != nil {
		log.Fatal(err)
	}
	for _, pc := range peeringConnections.Slice() {
		eq0, eq1, region0, region1 := pc.Ends()

		ui.Printf(
			"configuring VPC peering between %s %s %s and %s %s %s",
			eq0.Environment, eq0.Quality, region0,
			eq1.Environment, eq1.Quality, region1,
		)

		/*
			oldDirname := filepath.Join(
				terraform.RootModulesDirname,
				accounts.Network,
				eq0.Environment,
				"peering",
				fmt.Sprintf("%s-%s-%s-%s", eq0.Quality, region0, eq1.Quality, region1),
			)
			if err := terraform.Destroy(oldDirname); err != nil {
				log.Fatal(err)
			}
		*/

		dirname := filepath.Join(
			terraform.RootModulesDirname,
			accounts.Network,
			"peering",
			eq0.Environment,
			eq1.Environment,
			eq0.Quality,
			eq1.Quality,
			region0,
			region1,
		)

		file := terraform.NewFile()
		file.Push(terraform.Module{
			Arguments: map[string]terraform.Value{
				"accepter_environment":  terraform.Q(eq0.Environment),
				"accepter_quality":      terraform.Q(eq0.Quality),
				"requester_environment": terraform.Q(eq1.Environment),
				"requester_quality":     terraform.Q(eq1.Quality),
			},
			Label: terraform.Q("peering-connection"),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.ProviderAliasFor("accepter"):  terraform.ProviderAliasFor("accepter"),
				terraform.ProviderAliasFor("requester"): terraform.ProviderAliasFor("requester"),
			},
			Source: terraform.Q("../../../../../../../../../modules/peering-connection"),
		})
		if err := file.Write(filepath.Join(dirname, "main.tf")); err != nil {
			log.Fatal(err)
		}

		providersFile := terraform.NewFile()
		accepterProvider := terraform.ProviderFor(
			region1,
			roles.Arn(accountId, roles.NetworkAdministrator),
		)
		accepterProvider.Alias = "accepter"
		providersFile.Push(accepterProvider)
		requesterProvider := terraform.ProviderFor(
			region0,
			roles.Arn(accountId, roles.NetworkAdministrator),
		)
		requesterProvider.Alias = "requester"
		providersFile.Push(requesterProvider)
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			log.Fatal(err)
		}

		// The choice of region0 here is arbitrary.  Only one side
		// can store the Terraform state and region0 wins.
		if err := terraform.Root(dirname, region0); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Print("next, commit substrate.*, modules/peering-connection/, and root-modules/network/ to version control, then run `substrate bootstrap-deploy-account`")
}
