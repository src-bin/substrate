package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/ui"
)

func main() {

	// TODO offer the opportunity to use a subset of regions

	sess := awsutil.NewMasterSession("OrganizationReader")

	account, err := awsorgs.FindSpecialAccount(
		organizations.New(sess),
		"network",
	)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", account)

	netDoc, err := networks.ReadDocument()

	ui.Printf("bootstrapping the network in %d regions", len(awsutil.Regions()))
	for _, region := range awsutil.Regions() {
		ui.Spinf("bootstrapping the ops network in %s", region)

		sess := awsutil.NewSession(region)
		sess = sess.Copy(&aws.Config{
			Credentials: stscreds.NewCredentials(sess, fmt.Sprintf(
				"arn:aws:iam::%s:role/NetworkAdministrator",
				aws.StringValue(account.Id),
			)),
		})

		callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
		if err != nil {
			ui.Stop(err)
			continue
		}
		ui.Print(callerIdentity)

		net, err := netDoc.Next()
		if err != nil {
			log.Print(err)
			break
		}
		net.Region = region
		net.Special = "ops"

		net.IPv6 = "TODO"
		net.VPC = "TODO"

		ui.Stopf("%s %s %s", net.VPC, net.IPv4, net.IPv6)

		environments := []string{"development", "production"} // TODO
		qualities := []string{"alpha", "beta", "gamma"}       // TODO
		// TODO is it OK that we're creating networks in which there will never be IP address allocations?

		for _, environment := range environments {
			for _, quality := range qualities {
				continue
				ui.Spinf("bootstrapping the %s %s network in %s", environment, quality, region)

				// TODO

				net, err := netDoc.Next()
				if err != nil {
					log.Print(err)
					break
				}
				net.Environment = environment
				net.Quality = quality
				net.Region = region

				ui.Stop(net)
			}
		}

	}

	log.Print(netDoc, err)
	/*
		if err := netDoc.Write(); err != nil {
			log.Fatal(err)
		}
	*/

	// TODO peer everything together

}
