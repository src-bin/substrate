package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

func main() {
	domain := flag.String("domain", "", "domain for this new AWS account")
	environment := flag.String("environment", "", "environment for this new AWS account")
	quality := flag.String("quality", "", "quality for this new AWS account")
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	flag.Parse()
	if *domain == "" || *environment == "" || *quality == "" {
		ui.Fatal(`-domain="..." -environment="..." -quality"..." are required`)
	}
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	if !veqpDoc.Valid(*environment, *quality) {
		ui.Fatalf(`-environment="%s" -quality"%s" is not a valid environment and quality pair in your organization`, *environment, *quality)
	}

	sess, err := awssessions.InMasterAccount(roles.OrganizationAdministrator, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}

	ui.Spin("finding or creating the account")
	svc := organizations.New(sess)
	account, err := awsorgs.EnsureAccount(svc, *domain, *environment, *quality)
	if err != nil {
		log.Fatal(err)
	}
	if err := accounts.CheatSheet(svc); err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	admin.EnsureAdminRolesAndPolicies(sess)

	// Leave the user a place to put their own Terraform code that can be
	// shared between admin accounts of different qualities.
	/*
		if err := terraform.Scaffold(*domain, dirname); err != nil {
			log.Fatal(err)
		}
	*/

	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect every AWS region in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each region")
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, *domain, *environment, *quality, region)

		if err := terraform.Root(dirname, region); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}

		if *noApply {
			err = terraform.Plan(dirname)
		} else if *autoApprove {
			err = terraform.Apply(dirname)
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
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Printf(
		"next, commit modules/substrate/ and root-modules/%s/%s/%s/ to version control, then write Terraform code there to define the rest of your infrastructure or run substrate-create-account again for other domains, environments, and/or qualities",
		*domain,
		*environment,
		*quality,
	)

}
