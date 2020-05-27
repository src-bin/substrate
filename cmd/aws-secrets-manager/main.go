package main

import (
	"flag"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

func main() {

	name := flag.String("name", "", "name (in the UI) or ID (in the API) of the secret in AWS Secrets Manager")
	value := flag.String("value", "", "secret value to associate with -name (does not overwrite prior versions)")

	flag.Parse()
	if *name == "" {
		log.Fatal("-name is required")
	}

	sess := awssessions.NewSession(awssessions.Config{})

	stage := time.Now().Format(time.RFC3339)

	badRegions, goodRegions := []string{}, []string{}
	for _, region := range regions.Selected() {
		svc := secretsmanager.New(sess, &aws.Config{
			Region: aws.String(region),
		})
		if *value != "" {

			out, err := awssecretsmanager.PutSecretValue(svc, *name, stage, *value)
			if err != nil {
				badRegions = append(badRegions, region)
				ui.Printf("%s in %s: %s", *name, region, err)
				continue
			}
			ui.Printf("%+v", out)

		} else {

			out, err := awssecretsmanager.DescribeSecret(svc, *name)
			if err != nil {
				badRegions = append(badRegions, region)
				ui.Printf("%s in %s: %s", *name, region, err)
				continue
			}
			ui.Printf("%+v", out)

		}
		goodRegions = append(goodRegions, region)
	}

	if *value != "" {
		ui.Printf("the %s stage of %s is ready to use in the following regions:", stage, *name)
		for _, region := range goodRegions {
			ui.Printf("- %s", region)
		}
		ui.Printf("it is NOT ready to use in the following regions:")
		for _, region := range badRegions {
			ui.Printf("- %s", region)
		}
	}

}
