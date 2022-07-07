package main

import (
	"context"
	"flag"
	"strings"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
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
	stage := flag.String("stage", "", "identifier for this stage (or version) of the secret (to be provided when fetching it later)")
	value := flag.String("value", "", "secret value to associate with -name (does not overwrite prior versions)") // XXX do this with a prompt instead!
	flag.Usage = func() {
		ui.Print("Usage: aws-secrets-manager -name <name> [-principal <principal> [...]] [-stage <stage>] [-value <value>]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *name == "" {
		ui.Fatal("-name is required")
	}
	if *stage == "" {
		*stage = time.Now().Format(time.RFC3339)
	}

	principal := &policies.Principal{
		AWS: []string(*principals),
	}

	ctx := contextutil.WithValues(context.Background(), "aws-secrets-manager", "", "")

	cfg, err := awscfg.NewConfig(ctx)
	if err != nil {
		ui.Fatal(err)
	}

	badRegions, goodRegions := []string{}, []string{}
	for _, region := range regions.Selected() {
		cfg = cfg.Regional(region)

		if *value != "" {

			out, err := awssecretsmanager.EnsureSecret(
				ctx,
				cfg,
				*name,
				awssecretsmanager.Policy(principal),
				*stage,
				*value,
			)
			if err != nil {
				badRegions = append(badRegions, region)
				ui.Printf("%s in %s: %s", *name, region, err)
				continue
			}
			ui.Printf("%+v", out)

		} else if len(*principals) > 0 {

			out, err := awssecretsmanager.PutResourcePolicy(
				ctx,
				cfg,
				*name,
				awssecretsmanager.Policy(principal),
			)
			if err != nil {
				badRegions = append(badRegions, region)
				ui.Printf("%s in %s: %s", *name, region, err)
				continue
			}
			ui.Printf("%+v", out)

		} else {

			out, err := awssecretsmanager.DescribeSecret(ctx, cfg, *name)
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
			ui.Printf("the %s stage of %s is ready to use in the following regions:", *stage, *name)
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
