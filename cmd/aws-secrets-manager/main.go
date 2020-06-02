package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

type StringSliceFlag []string

func (ssf *StringSliceFlag) String() string {
	return strings.Join(*ssf, ", ")
}

func (ssf *StringSliceFlag) Set(s string) error {
	*ssf = append(*ssf, s)
	return nil
}

func main() {

	name := flag.String("name", "", "name (in the UI) or ID (in the API) of the secret in AWS Secrets Manager")
	principals := &StringSliceFlag{}
	flag.Var(principals, "principal", "principal ARN to be allowed to GetSecretValue (if any are provided, the secret's policy will be updated to allow exactly and only those principals given)")
	value := flag.String("value", "", "secret value to associate with -name (does not overwrite prior versions)")

	flag.Parse()
	if *name == "" {
		log.Fatal("-name is required")
	}

	doc := &policies.Document{
		Statement: []policies.Statement{
			policies.Statement{
				Action: []string{"secretsmanager:GetSecretValue"},
				Principal: &policies.Principal{
					AWS: []string(*principals),
				},
				Resource: []string{"*"},
			},
		},
	}

	sess := awssessions.Must(awssessions.NewSession(awssessions.Config{}))

	stage := time.Now().Format(time.RFC3339)

	badRegions, goodRegions := []string{}, []string{}
	for _, region := range regions.Selected() {

		svc := secretsmanager.New(sess, &aws.Config{
			Region: aws.String(region),
		})

		if *value != "" {

			out, err := awssecretsmanager.EnsureSecret(
				svc,
				*name,
				doc,
				stage,
				*value,
			)
			if err != nil {
				badRegions = append(badRegions, region)
				ui.Printf("%s in %s: %s", *name, region, err)
				continue
			}
			ui.Printf("%+v", out)

		} else if len(*principals) > 0 {

			out, err := awssecretsmanager.PutResourcePolicy(svc, *name, doc)
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

	if len(*principals) > 0 || *value != "" {
		if len(*principals) > 0 {
			ui.Printf("the resource policy for %s is updated in the following regions:", *name)
		} else if *value != "" {
			ui.Printf("the %s stage of %s is ready to use in the following regions:", stage, *name)
		}
		for _, region := range goodRegions {
			ui.Printf("- %s", region)
		}
		if len(badRegions) > 0 {
			ui.Printf("but NOT in these regions:")
			for _, region := range badRegions {
				ui.Printf("- %s", region)
			}
		}
	}

}
