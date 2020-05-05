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
)

func main() {

	environments := []string{"development", "production"} // TODO
	qualities := []string{"alpha", "beta", "gamma"}       // TODO
	// TODO is it OK that we're creating networks in which there will never be IP address allocations?

	// TODO offer the opportunity to use a subset of regions

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

	d, err := networks.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", d)

	ops := terraform.NewBlocks()

	ui.Printf("bootstrapping the network in %d regions", len(awsutil.Regions()))
	for _, region := range awsutil.Regions() {
		ui.Spinf("finding or assigning an IP address range to the ops network in %s", region)

		n, err := d.Ensure(&networks.Network{
			Region:  region,
			Special: "ops",
		})
		if err != nil {
			log.Fatal(err)
		}
		//log.Printf("%+v", n)

		ops.Push(terraform.VPC{
			CidrBlock: n.IPv4.String(),
			Label:     fmt.Sprintf("ops-%s", region),
		})

		ui.Stop(n.IPv4)

		for _, environment := range environments {
			for _, quality := range qualities {
				continue
				ui.Spinf(
					"finding or assigning an IP address range to the %s %s network in %s",
					environment,
					quality,
					region,
				)

				n, err := d.Ensure(&networks.Network{
					Environment: environment,
					Quality:     quality,
					Region:      region,
				})
				if err != nil {
					log.Fatal(err)
				}
				//log.Printf("%+v", n)

				ui.Stop(n)
			}
		}

	}

	// TODO peer everything together

	// Write to substrate.networks.json once more so that, even if no changes
	// were made, formatting changes and SubstrateVersion are changed.
	if err := d.Write(); err != nil {
		log.Fatal(err)
	}

	providers := terraform.Provider{
		AccountId:   aws.StringValue(account.Id),
		RoleName:    "NetworkAdministrator",
		SessionName: "Terraform",
	}.AllRegions()
	for _, quality := range qualities {
		if err := providers.Write(path.Join("network-account", quality, "providers.tf")); err != nil {
			log.Fatal(err)
		}
	}

	if err := ops.Write(path.Join("network-account", qualities[0], "ops.tf")); err != nil {
		log.Fatal(err)
	}

	if err := terraform.Fmt(); err != nil {
		log.Fatal(err)
	}

	// TODO assume the NetworkAdministrator role in each region and apply the generated Terraform code

}
