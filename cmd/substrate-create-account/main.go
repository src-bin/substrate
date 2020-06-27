package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

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

	dirname := fmt.Sprintf("%s-%s-%s-account", *domain, *environment, *quality)

	// Write (or rewrite) some Terraform providers to make everything usable.
	providersFile := terraform.NewFile()
	providersFile.PushAll(terraform.Provider{
		AccountId:   aws.StringValue(account.Id),
		RoleName:    roles.Administrator,
		SessionName: "Terraform",
	}.AllRegionsAndGlobal())
	networkAccount, err := awsorgs.FindSpecialAccount(svc, accounts.Network)
	if err != nil {
		log.Fatal(err)
	}
	providersFile.PushAll(terraform.Provider{
		AccountId:   aws.StringValue(networkAccount.Id),
		AliasSuffix: "network",
		RoleName:    roles.Auditor,
		SessionName: "Terraform",
	}.AllRegions())
	if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
		log.Fatal(err)
	}

	// Generate the files and directory structure needed to get the user
	// started writing their own Terraform code.
	if err := terraform.Scaffold(*domain, dirname); err != nil {
		log.Fatal(err)
	}

	// Format all the Terraform code you can possibly find.
	if err := terraform.Fmt(); err != nil {
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
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	} else {
		if err := terraform.Apply(dirname); err != nil {
			log.Fatal(err)
		}
	}

	// TODO more?

	ui.Printf(
		"next, commit %s-%s-%s-account/ to version control, then write Terraform code there to define the rest of your infrastructure or run substrate-create-account again for other domains, environments, and/or qualities",
		*domain,
		*environment,
		*quality,
	)

}
