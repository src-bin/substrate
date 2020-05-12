package main

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/availabilityzones"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const (
	EnvironmentsFilename = "substrate.Environments"
	QualitiesFilename    = "substrate.Qualities"
	TerraformDirname     = "network-account"
)

func main() {

	// Gather the definitive list of Environments and Qualities first.
	environments, err := ui.EditFile(
		EnvironmentsFilename,
		"the following Environments are currently valid in your Substrate-managed infrastructure:",
		"list all your Environments, one per line, in order of progression from e.g. development through e.g. production",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using Environments %s", strings.Join(environments, ", "))
	qualities, err := ui.EditFile(
		QualitiesFilename,
		"the following Qualities are currently valid in your Substrate-managed infrastructure:",
		`list all your Qualities, one per line, in order from least to greatest quality (Substrate recommends "alpha", "beta", and "gamma")`,
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(qualities) < 2 {
		ui.Print(`you must define at least two Qualities (and Substrate recommends "alpha", "beta", and "gamma")`)
		return
	}
	ui.Printf("using Qualities %s", strings.Join(qualities, ", "))

	// Combine all Environments and Qualities.  If a given combination doesn't
	// appear in substrate.ValidEnvironmentQualityPairs.json then offer its
	// inclusion before validating the final document.
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	for _, environment := range environments {
		for _, quality := range qualities {
			if !veqpDoc.Valid(environment, quality) {
				ok, err := ui.Confirmf(`do you want to allow %s-Quality infrastructure in your %s Environment?`, quality, environment)
				if err != nil {
					log.Fatal(err)
				}
				if ok {
					veqpDoc.Ensure(environment, quality)
				}
			}
		}
	}
	if err := veqpDoc.Validate(environments, qualities); err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", veqpDoc)

	// Select or confirm which regions to use.
	if _, err := regions.Select(); err != nil {
		log.Fatal(err)
	}

	netDoc, err := networks.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", netDoc)

	// Make changes to the ops network more testable by designating one region
	// as the guinea pig.
	var alphaRegion string
	if n := netDoc.Find(&networks.Network{Quality: qualities[0], Special: "ops"}); n == nil {
		ui.Printf(
			"most of your ops account will be designated %s-Quality (this controls the order in which Terraform changes are applied) but you should designate one region to be %s-Quality so changes may be tested before affecting your entire ops network",
			qualities[1],
			qualities[0],
		)
		region, err := ui.Promptf("what region's ops network should be designated %s-Quality?", qualities[0])
		if err != nil {
			log.Fatal(err)
		}
		if !regions.IsBlacklisted(region) {
			log.Fatalf("%s is is blacklisted in this Substrate installation", region)
		}
		if !regions.IsRegion(region) {
			log.Fatalf("%s is not an AWS region", region)
		}
		alphaRegion = region
	} else {
		alphaRegion = n.Region
	}
	ui.Printf(
		"marking the ops network in %s as %s-Quality (other regions will be %s-Quality)",
		alphaRegion,
		qualities[0],
		qualities[1],
	)

	sess := awssessions.AssumeRoleMaster(
		awssessions.NewSession(awssessions.Config{}),
		roles.OrganizationReader,
	)
	account, err := awsorgs.FindSpecialAccount(
		organizations.New(sess),
		accounts.Network,
	)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", account)
	sess = awssessions.AssumeRole(
		awssessions.NewSession(awssessions.Config{}),
		aws.StringValue(account.Id),
		"NetworkAdministrator",
	)

	ui.Printf("bootstrapping the ops network in %d regions", len(regions.Selected()))
	blockses := []terraform.Blocks{terraform.NewBlocks(), terraform.NewBlocks()}
	for _, region := range regions.Selected() {
		ui.Spinf("finding or assigning an IP address range to the ops network in %s", region)

		i := 1
		if region == alphaRegion {
			i = 0
		}

		n, err := netDoc.Ensure(&networks.Network{
			Quality: qualities[i],
			Region:  region,
			Special: "ops",
		})
		if err != nil {
			log.Fatal(err)
		}
		//log.Printf("%+v", n)

		vpc := terraform.VPC{
			CidrBlock: terraform.Q(n.IPv4.String()),
			Provider:  terraform.ProviderAliasFor(region),
			Tags: terraform.Tags{
				Name:    fmt.Sprintf("ops-%s", region),
				Quality: qualities[i],
				Special: "ops",
			},
		}
		blockses[i].Push(vpc)

		azs, err := availabilityzones.Select(sess, region, 3)
		if err != nil {
			log.Fatal(err)
		}
		for j, az := range azs {
			blockses[i].Push(terraform.Subnet{
				AvailabilityZone:    terraform.Q(az),
				CidrBlock:           vpc.CidrsubnetIPv4(4, j+1),
				IPv6CidrBlock:       vpc.CidrsubnetIPv6(8, j+1),
				MapPublicIPOnLaunch: true,
				Provider:            terraform.ProviderAliasFor(region),
				Tags: terraform.Tags{
					Quality: qualities[i],
					Special: "ops",
				},
				VpcId: terraform.Uf("aws_vpc.ops-%s.id", region),
			})
			blockses[i].Push(terraform.Subnet{
				AvailabilityZone: terraform.Q(az),
				CidrBlock:        vpc.CidrsubnetIPv4(2, j+1),
				IPv6CidrBlock:    vpc.CidrsubnetIPv6(8, j+5),
				Provider:         terraform.ProviderAliasFor(region),
				Tags: terraform.Tags{
					Quality: qualities[i],
					Special: "ops",
				},
				VpcId: terraform.Uf("aws_vpc.ops-%s.id", region),
			})
		}

		ui.Stop(n.IPv4)
	}
	for i := 0; i < len(blockses); i++ {
		if err := blockses[i].Write(path.Join(TerraformDirname, "ops", qualities[i], "vpc.tf")); err != nil {
			log.Fatal(err)
		}
	}

	ui.Printf("bootstrapping networks for every Environment and Quality in %d regions", len(regions.Selected()))
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		blocks := terraform.NewBlocks()

		for _, region := range regions.Selected() {
			ui.Spinf(
				"finding or assigning an IP address range to the %s %s network in %s",
				eq.Environment,
				eq.Quality,
				region,
			)

			n, err := netDoc.Ensure(&networks.Network{
				Environment: eq.Environment,
				Quality:     eq.Quality,
				Region:      region,
			})
			if err != nil {
				log.Fatal(err)
			}
			//log.Printf("%+v", n)

			/*
				blocks.Push(terraform.VPC{
					CidrBlock: n.IPv4.String(),
					Provider:  terraform.ProviderAliasFor(region),
				})
			*/

			ui.Stop(n.IPv4)
		}

		if err := blocks.Write(path.Join(TerraformDirname, eq.Environment, eq.Quality, "vpc.tf")); err != nil {
			log.Fatal(err)
		}

	}

	// TODO peer everything together and setup subnet sharing

	// Write to substrate.Networks.json once more so that, even if no changes
	// were made, formatting changes and SubstrateVersion are changed.
	if err := netDoc.Write(); err != nil {
		log.Fatal(err)
	}

	// Write some Terraform providers to make everything usable.
	providers := terraform.Provider{
		AccountId:   aws.StringValue(account.Id),
		RoleName:    roles.NetworkAdministrator,
		SessionName: "Terraform",
	}.AllRegions()
	for i := 0; i < 2; i++ {
		if err := providers.Write(path.Join(TerraformDirname, "ops", qualities[i], "providers.tf")); err != nil {
			log.Fatal(err)
		}
	}
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		if err := providers.Write(path.Join(TerraformDirname, eq.Environment, eq.Quality, "providers.tf")); err != nil {
			log.Fatal(err)
		}
	}

	// Format all the Terraform code you can possibly find.
	if err := terraform.Fmt(); err != nil {
		log.Fatal(err)
	}

	// Ensure the VPCs-per-region service quota isn't going to get in the way.
	ui.Print("raising the VPCs-per-region service quota in all regions (this could take days, unfortunately; this program is safe to re-run)")
	if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
		sess,
		"L-F678F1CE",
		"vpc",
		float64(len(netDoc.FindAll(&networks.Network{Region: "us-west-2"}))+ // networks for existing (Environment, Quality) pairs
			len(qualities)+ // plus room to add another Environment
			1), // plus the ops network
	); err != nil {
		log.Fatal(err)
	}

	// Generate a Makefile in each root Terraform module then apply the generated
	// Terraform code.  Start with the ops networks, then move on to the
	// Environments, all Quality-by-Quality with a pause in between.
	// TODO confirmation between steps
	for i := 0; i < 2; i++ {
		dirname := path.Join(TerraformDirname, "ops", qualities[i])
		if err := terraform.Makefile(dirname); err != nil {
			log.Fatal(err)
		}
		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}
		if err := terraform.Apply(dirname); err != nil {
			log.Fatal(err)
		}
	}
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		dirname := path.Join(TerraformDirname, eq.Environment, eq.Quality)
		if err := terraform.Makefile(dirname); err != nil {
			log.Fatal(err)
		}
		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}
		if err := terraform.Plan(dirname); err != nil {
			log.Fatal(err)
		}
	}

}
