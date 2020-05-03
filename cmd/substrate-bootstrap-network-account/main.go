package main

import (
	"log"

	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/ui"
)

func main() {

	// TODO offer the opportunity to use a subset of regions

	netDoc, err := networks.ReadDocument()
	log.Print(netDoc, err)

	ui.Printf("bootstrapping the network in %d regions", len(awsutil.Regions()))
	for _, region := range awsutil.Regions() {

		ui.Spinf("bootstrapping the ops network in %s", region)
		/*
			sess := awsutil.NewSession(region)

			callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
			if err != nil {
				ui.Stop(err)
				continue
			}
			ui.Print(callerIdentity)
		*/

		net, err := netDoc.Next()
		if err != nil {
			log.Print(err)
			break
		}
		net.Region = region
		net.Special = "ops"

		ui.Stop(net)

		environments := []string{"development", "staging", "production"} // TODO
		qualities := []string{"alpha", "beta", "gamma"}                  // TODO

		for _, environment := range environments {
			for _, quality := range qualities {
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
