package main

import (
	"flag"
	"fmt"
	"log"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

func main() {
	domain := flag.String("domain", "", "domain for this new AWS account")
	environment := flag.String("environment", "", "environment for this new AWS account")
	quality := flag.String("quality", "", "quality for this new AWS account")
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
	account, err := awsorgs.EnsureAccount(organizations.New(sess), *domain, *environment, *quality)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", account.Id)
	log.Printf("%+v", account)

	admin.EnsureAdministratorRolesAndPolicies(sess)

	dirname := fmt.Sprintf("%s-%s-%s-account", *domain, *environment, *quality)

	// Write (or rewrite) some Terraform providers to make everything usable.
	providers := terraform.Provider{
		AccountId:   aws.StringValue(account.Id),
		RoleName:    roles.Administrator,
		SessionName: "Terraform",
	}.AllRegions()
	if err := providers.Write(path.Join(dirname, "providers.tf")); err != nil {
		log.Fatal(err)
	}

	// Generate a Makefile in the root Terraform module then apply the generated
	// Terraform code.
	if err := terraform.Root(dirname, awssessions.Must(awssessions.InSpecialAccount(
		accounts.Deploy,
		roles.DeployAdministrator,
		awssessions.Config{},
	))); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Init(dirname); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Apply(dirname); err != nil {
		log.Fatal(err)
	}

	// TODO more?

	ui.Printf(
		"next, commit %s-%s-%s-account/ to version control, then write Terraform code there to define the rest of your infrastructure or run substrate-create-account again for other domains, environments, and/or qualities",
		*domain,
		*environment,
		*quality,
	)

}
