package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/availabilityzones"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const (
	EnvironmentsFilename = "substrate.environments"
	QualitiesFilename    = "substrate.qualities"
)

func main() {
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	flag.Parse()

	sess, err := awssessions.InSpecialAccount(accounts.Network, roles.NetworkAdministrator, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Gather the definitive list of environments and qualities first.
	environments, err := ui.EditFile(
		EnvironmentsFilename,
		"the following environments are currently valid in your Substrate-managed infrastructure:",
		`list all your environments, one per line, in order of progression from e.g. development through e.g. production; your list MUST include "admin"`,
	)
	if err != nil {
		log.Fatal(err)
	}
	found := false
	for _, environment := range environments {
		found = found || environment == "admin"
	}
	if !found {
		ui.Fatal(`you must include "admin" in your list of environments`)
	}
	ui.Printf("using environments %s", strings.Join(environments, ", "))
	qualities, err := ui.EditFile(
		QualitiesFilename,
		"the following qualities are currently valid in your Substrate-managed infrastructure:",
		`list all your qualities, one per line, in order from least to greatest quality (Substrate recommends "alpha", "beta", and "gamma")`,
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(qualities) == 0 {
		ui.Fatal("you must name at least one quality")
	}
	ui.Printf("using qualities %s", strings.Join(qualities, ", "))

	// Combine all environments and qualities.  If a given combination doesn't
	// appear in substrate.valid-environment-quality-pairs.json then offer its
	// inclusion before validating the final document.
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	ui.Print("you currently allow the following combinations of environment and quality in your Substrate-managed infrastructure:")
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		ui.Printf("\t%-12s %s", eq.Environment, eq.Quality)
	}
	ok, err := ui.Confirm("is this correct? (yes/no)")
	if err != nil {
		log.Fatal(err)
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
	if err := veqpDoc.Validate(environments, qualities); err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", veqpDoc)

	if _, err := regions.Select(); err != nil {
		log.Fatal(err)
	}

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
				Label:    terraform.Q("current"),
				Provider: terraform.GlobalProviderAlias,
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
				Provider:  terraform.ProviderAliasFor(region),
				Tags:      tags,
			}
			file.Push(vpc)
			vpcAccoutrements(sess, region, org, vpc, file)
			if err := file.Write(filepath.Join(dirname, "main.tf")); err != nil {
				log.Fatal(err)
			}

		}
	}

	// Write to substrate.admin-networks.json and substrate.networks.json once
	// more so that, even if no changes were made, formatting changes and
	// SubstrateVersion are changed.
	if err := adminNetDoc.Write(); err != nil {
		log.Fatal(err)
	}
	if err := netDoc.Write(); err != nil {
		log.Fatal(err)
	}

	// Ensure the VPCs-per-region service quota and a few others that  isn't going to get in the way.
	ui.Print("raising the VPC, Internet, Egress-Only Internet, and NAT Gateway, and EIP service quotas in all your regions (this could take days, unfortunately; this program is safe to re-run)")
	desiredValue := float64(len(netDoc.FindAll(&networks.Network{Region: regions.Selected()[0]})) + // networks for existing (environment, quality) pairs
		len(qualities) + // plus room to add another environment
		1) // plus the default VPC
	for _, quota := range [][2]string{
		{"L-F678F1CE", "vpc"}, // VPCs per region
		{"L-45FE3B85", "vpc"}, // Egress-Only Internet Gateways per region
		{"L-A4707A72", "vpc"}, // Internet Gateways per region
		{"L-FE5A380F", "vpc"}, // NAT Gateways per availability zone
		// {"L-0263D0A3", "ec2"}, // EIPs per VPC
	} {
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			sess,
			quota[0],
			quota[1],
			desiredValue,
		); err != nil {
			log.Fatal(err)
		}
	}

	// Generate a Makefile in each root Terraform module then apply the generated
	// Terraform code.  Start with the ops networks, then move on to the
	// environments, all quality-by-quality with a pause in between.
	if !*noApply {
		ui.Print("this tool can affect multiple environments and qualities in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each enviornment and quality")
	}
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region := range regions.Selected() {
			dirname := filepath.Join(terraform.RootModulesDirname, accounts.Network, eq.Environment, eq.Quality, region)

			// TODO setup global and regional modules just like in other accounts

			providersFile := terraform.NewFile()
			providersFile.Push(terraform.ProviderFor(
				region,
				roles.Arn(accountId, roles.NetworkAdministrator),
			))
			providersFile.Push(terraform.GlobalProvider(
				roles.Arn(accountId, roles.NetworkAdministrator),
			))
			networkAccount, err := awsorgs.FindSpecialAccount(organizations.New(awssessions.Must(awssessions.InMasterAccount(
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
				ok, err := ui.Confirmf("apply Terraform changes in %s? (yes/no)", dirname)
				if err != nil {
					log.Fatal(err)
				}
				if ok {
					err = terraform.Apply(dirname)
				}
			}
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Print("next, commit substrate.* and root-modules/network/ to version control, then run substrate-bootstrap-deploy-account")
}

func vpcAccoutrements(
	sess *session.Session,
	region string,
	org terraform.Organization,
	vpc terraform.VPC,
	file *terraform.File,
) {
	hasPrivateSubnets := vpc.Tags.Environment != "admin"

	// Accept the default Network ACL until we need to do otherwise.

	// TODO manage the default security group to ensure it has no rules.

	// Accept the default DHCP option set until we need to do otherwise.

	// A resource share for the subnets to reference, shared org-wide.
	rs := terraform.ResourceShare{
		Label:    terraform.Label(vpc.Tags),
		Provider: vpc.Provider,
		Tags:     vpc.Tags,
	}
	file.Push(rs)
	file.Push(terraform.PrincipalAssociation{
		Label:            terraform.Label(vpc.Tags),
		Principal:        terraform.U(org.Ref(), ".arn"),
		Provider:         vpc.Provider,
		ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
	})

	// IPv4 and IPv6 Internet Gateways for the public subnets.
	igw := terraform.InternetGateway{
		Label:    terraform.Label(vpc.Tags),
		Provider: vpc.Provider,
		Tags:     vpc.Tags,
		VpcId:    terraform.U(vpc.Ref(), ".id"),
	}
	file.Push(igw)
	file.Push(terraform.Route{
		DestinationIPv4:   terraform.Q("0.0.0.0/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(vpc.Tags, "public-internet-ipv4"),
		Provider:          vpc.Provider,
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	})
	file.Push(terraform.Route{
		DestinationIPv6:   terraform.Q("::/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(vpc.Tags, "public-internet-ipv6"),
		Provider:          vpc.Provider,
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	})

	// IPv6 Egress-Only Internet Gateway for the private subnets.  (The IPv4
	// NAT Gateway comes later because it's per-subnet.  That is also where
	// this Egress-Only Internat Gateway is associated with the route table.)
	egw := terraform.EgressOnlyInternetGateway{
		Label:    terraform.Label(vpc.Tags),
		Provider: vpc.Provider,
		Tags:     vpc.Tags,
		VpcId:    terraform.U(vpc.Ref(), ".id"),
	}
	if hasPrivateSubnets {
		file.Push(egw)
	}

	// Create a public and private subnet in each of (up to, and the newest)
	// three availability zones in the region.
	azs, err := availabilityzones.Select(sess, region, 3)
	if err != nil {
		log.Fatal(err)
	}
	for i, az := range azs {
		tags := terraform.Tags{
			AvailabilityZone: az,
			Environment:      vpc.Tags.Environment,
			Quality:          vpc.Tags.Quality,
			Region:           region,
			Special:          vpc.Tags.Special,
		}

		// Public subnet, shared org-wide.
		bits := 2
		if hasPrivateSubnets {
			bits = 4
		}
		s := terraform.Subnet{
			AvailabilityZone:    terraform.Q(az),
			CidrBlock:           vpc.CidrsubnetIPv4(bits, i+1),
			IPv6CidrBlock:       vpc.CidrsubnetIPv6(8, i+1),
			Label:               terraform.Label(tags, "public"),
			MapPublicIPOnLaunch: true,
			Provider:            vpc.Provider,
			Tags:                tags,
			VpcId:               terraform.U(vpc.Ref(), ".id"),
		}
		s.Tags.Connectivity = "public"
		s.Tags.Name = vpc.Tags.Name + "-public-" + az
		file.Push(s)
		file.Push(terraform.ResourceAssociation{
			Label:            s.Label,
			Provider:         vpc.Provider,
			ResourceArn:      terraform.U(s.Ref(), ".arn"),
			ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
		})

		// Explicitly associate the public subnets with the main routing table.
		file.Push(terraform.RouteTableAssociation{
			Label:        s.Label,
			Provider:     vpc.Provider,
			RouteTableId: terraform.U(vpc.Ref(), ".default_route_table_id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		})

		if !hasPrivateSubnets {
			continue
		}

		// Private subnet, also shared org-wide.
		s = terraform.Subnet{
			AvailabilityZone: terraform.Q(az),
			CidrBlock:        vpc.CidrsubnetIPv4(2, i+1),
			IPv6CidrBlock:    vpc.CidrsubnetIPv6(8, i+0x81),
			Label:            terraform.Label(tags, "private"),
			Provider:         vpc.Provider,
			Tags:             tags,
			VpcId:            terraform.U(vpc.Ref(), ".id"),
		}
		s.Tags.Connectivity = "private"
		s.Tags.Name = vpc.Tags.Name + "-private-" + az
		file.Push(s)
		file.Push(terraform.ResourceAssociation{
			Label:            s.Label,
			Provider:         vpc.Provider,
			ResourceArn:      terraform.U(s.Ref(), ".arn"),
			ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
		})

		// Private subnets need their own routing tables to keep their NAT
		// Gateway traffic zonal.
		rt := terraform.RouteTable{
			Label:    s.Label,
			Provider: vpc.Provider,
			Tags:     s.Tags,
			VpcId:    terraform.U(vpc.Ref(), ".id"),
		}
		file.Push(rt)
		file.Push(terraform.RouteTableAssociation{
			Label:        s.Label,
			Provider:     vpc.Provider,
			RouteTableId: terraform.U(rt.Ref(), ".id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		})

		// NAT Gateway for this private subnet.
		eip := terraform.EIP{
			InternetGatewayRef: igw.Ref(),
			Label:              terraform.Label(tags),
			Provider:           vpc.Provider,
			Tags:               tags,
		}
		eip.Tags.Name = vpc.Tags.Name + "-nat-gateway-" + az
		file.Push(eip)
		ngw := terraform.NATGateway{
			Label:    terraform.Label(tags),
			Provider: vpc.Provider,
			SubnetId: terraform.U(s.Ref(), ".id"),
			Tags:     tags,
		}
		ngw.Tags.Name = vpc.Tags.Name + "-" + az
		file.Push(ngw)
		file.Push(terraform.Route{
			DestinationIPv4: terraform.Q("0.0.0.0/0"),
			Label:           terraform.Label(tags),
			NATGatewayId:    terraform.U(ngw.Ref(), ".id"),
			Provider:        vpc.Provider,
			RouteTableId:    terraform.U(rt.Ref(), ".id"),
		})

		// Associate the VPC's Egress-Only Internet Gateway for IPv6 traffic.
		file.Push(terraform.Route{
			DestinationIPv6:             terraform.Q("::/0"),
			EgressOnlyInternetGatewayId: terraform.U(egw.Ref(), ".id"),
			Label:                       terraform.Label(s.Tags, "private-internet-ipv6"),
			Provider:                    vpc.Provider,
			RouteTableId:                terraform.U(rt.Ref(), ".id"),
		})

	}

}
