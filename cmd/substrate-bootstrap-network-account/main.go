package main

import (
	"fmt"
	"log"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const (
	EnvironmentsFilename = "substrate.Environments"
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
	// TODO qualities

	netDoc, err := networks.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", netDoc)

	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", veqpDoc)

	veqpDoc.Ensure(veqp.EnvironmentQualityPair{"development", "alpha"})
	veqpDoc.Ensure(veqp.EnvironmentQualityPair{"development", "beta"})
	veqpDoc.Ensure(veqp.EnvironmentQualityPair{"production", "beta"})
	veqpDoc.Ensure(veqp.EnvironmentQualityPair{"production", "gamma"})
	// TODO if the JSON file doesn't exist, prime a file with pairwise "Environment Quality" lines and ask the user to remove the ones they don't want to allow
	qualities := veqpDoc.Qualities()

	if len(qualities) < 2 {
		ui.Print(`you must define at least two Qualities (and Substrate recommends "alpha", "beta", and "gamma")`)
		return
	}

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
		if !awsutil.IsRegion(region) {
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

	// TODO offer the opportunity to use a subset of regions by storing a list in substrate.regions and only allocating to those

	sess := awssessions.AssumeRoleMaster(
		awssessions.NewSession(awssessions.Config{}),
		"OrganizationReader",
	)
	account, err := awsorgs.FindSpecialAccount(
		organizations.New(sess),
		"network",
	)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", account)

	ui.Printf("bootstrapping the ops network in %d regions", len(awsutil.Regions()))
	blockses := []terraform.Blocks{terraform.NewBlocks(), terraform.NewBlocks()}
	for _, region := range awsutil.Regions() {
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

		blockses[i].Push(terraform.VPC{
			CidrBlock: n.IPv4.String(),
			Label:     fmt.Sprintf("ops-%s", region),
			Provider:  terraform.ProviderAliasFor(region),
		})

		ui.Stop(n.IPv4)
	}
	for i := 0; i < len(blockses); i++ {
		if err := blockses[i].Write(path.Join(TerraformDirname, "ops", qualities[i], "vpc.tf")); err != nil {
			log.Fatal(err)
		}
	}

	ui.Printf("bootstrapping networks for every Environment and Quality in %d regions", len(awsutil.Regions()))
	for _, environment := range environments {
		for _, quality := range qualities {
			blocks := terraform.NewBlocks()

			for _, region := range awsutil.Regions() {
				ui.Spinf(
					"finding or assigning an IP address range to the %s %s network in %s",
					environment,
					quality,
					region,
				)

				n, err := netDoc.Ensure(&networks.Network{
					Environment: environment,
					Quality:     quality,
					Region:      region,
				})
				if err != nil {
					log.Fatal(err)
				}
				//log.Printf("%+v", n)

				blocks.Push(terraform.VPC{
					CidrBlock: n.IPv4.String(),
					Label:     fmt.Sprintf("%s-%s-%s", environment, quality, region),
					Provider:  terraform.ProviderAliasFor(region),
				})

				ui.Stop(n.IPv4)
			}

			if err := blocks.Write(path.Join(TerraformDirname, environment, quality, "vpc.tf")); err != nil {
				log.Fatal(err)
			}

		}

	}

	// TODO peer everything together and setup subnet sharing

	// Write to substrate.networks.json once more so that, even if no changes
	// were made, formatting changes and SubstrateVersion are changed.
	if err := netDoc.Write(); err != nil {
		log.Fatal(err)
	}

	// Write some Terraform providers to make everything usable.
	providers := terraform.Provider{
		AccountId:   aws.StringValue(account.Id),
		RoleName:    "NetworkAdministrator",
		SessionName: "Terraform",
	}.AllRegions()
	for _, environment := range environments {
		for _, quality := range qualities {
			if err := providers.Write(path.Join(TerraformDirname, environment, quality, "providers.tf")); err != nil {
				log.Fatal(err)
			}
		}
	}

	if err := terraform.Fmt(); err != nil {
		log.Fatal(err)
	}

	// TODO assume the NetworkAdministrator role in each region and apply the generated Terraform code

}
