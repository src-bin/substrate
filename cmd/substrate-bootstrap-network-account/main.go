package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/ui"
)

func main() {

	// TODO offer the opportunity to use a subset of regions

	sess := awssessions.AssumeRoleMaster(
		awssessions.NewSession(awssessions.Config()),
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

	ui.Printf("bootstrapping the network in %d regions", len(awsutil.Regions()))
	for _, region := range awsutil.Regions() {
		ui.Spinf("bootstrapping the ops network in %s", region)

		n, err := d.Next(&networks.Network{
			Region:  region,
			Special: "ops",
		}) // TODO need to search the document for existing matching networks
		if err != nil {
			log.Fatal(err)
		}

		ui.Stopf("%s %s %s", n.VPC, n.IPv4, n.IPv6)

		environments := []string{"development", "production"} // TODO
		qualities := []string{"alpha", "beta", "gamma"}       // TODO
		// TODO is it OK that we're creating networks in which there will never be IP address allocations?

		for _, environment := range environments {
			for _, quality := range qualities {
				continue
				ui.Spinf("bootstrapping the %s %s network in %s", environment, quality, region)

				// TODO

				n, err := d.Next(&networks.Network{
					Environment: environment,
					Quality:     quality,
					Region:      region,
				}) // TODO need to search the document for existing matching networks
				if err != nil {
					log.Fatal(err)
				}

				ui.Stop(n)
			}
		}

	}

	// TODO peer everything together

	sess := awssessions.AssumeRole(awssessions.NewSession(
		awssessions.Config().WithRegion(region)),
		aws.StringValue(account.Id),
		"NetworkAdministrator",
	)
	svc := ec2.New(sess)

	// TODO apply generated Terraform code

}
