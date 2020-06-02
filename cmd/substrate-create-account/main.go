package main

import (
	"flag"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

func main() {
	domain := flag.String("domain", "", "Domain for this new AWS account")
	environment := flag.String("environment", "", "Environment for this new AWS account")
	quality := flag.String("quality", "", "Quality for this new AWS account")
	flag.Parse()
	if *domain == "" || *environment == "" || *quality == "" {
		ui.Fatal(`-domain="..." -environment="..." -quality"..." are required`)
	}
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	if !veqpDoc.Valid(*environment, *quality) {
		ui.Fatalf(`-environment="%s" -quality"%s" is not a valid Environment and Quality pair in your organization`, *environment, *quality)
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

	// Allow any role in this account to assume the OrganizationReader role in
	// the master account.
	svc := iam.New(sess)
	role, err := awsiam.GetRole(svc, roles.OrganizationReader)
	if err != nil {
		log.Fatal(err)
	}
	if err := role.AddPrincipal(svc, &policies.Principal{
		AWS: []string{aws.StringValue(account.Id)},
	}); err != nil {
		log.Fatal(err)
	}

	// TODO more?

	// TODO next steps?

}
