package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const OktaMetadataFilename = "substrate.okta.xml"

func main() {
	oktaMetadataPathname := flag.String("okta-metadata", OktaMetadataFilename, "pathname of a file containing your Okta SAML provider metadata")
	quality := flag.String("quality", "", "Quality for this new AWS account")
	flag.Parse()
	if *quality == "" {
		ui.Print(`-quality"..." is required`)
		os.Exit(1)
	}
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	if !veqpDoc.Valid("admin", *quality) {
		ui.Printf(`-quality"%s" is not a valid Quality for an admin account in your organization`, *quality)
		os.Exit(1)
	}

	lines, err := ui.EditFile(
		*oktaMetadataPathname,
		"here is your current identity provider metadata XML:",
		"paste your identity provider metadata XML from Okta",
	)
	if err != nil {
		log.Fatal(err)
	}
	metadata := strings.Join(lines, "\n") + "\n" // ui.EditFile is line-oriented but this instance isn't

	sess, err := awssessions.InMasterAccount(roles.OrganizationAdministrator, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Ensure the account exists.
	ui.Spin("finding or creating the admin account")
	account, err := awsorgs.EnsureAccount(organizations.New(sess), accounts.Admin, accounts.Admin, *quality)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	okta(sess, account, metadata)

	// Allow only the Administrator role in this account to assume the
	// OrganizationAdministrator role in the master account.  Allow any role
	// in this account to assume the OrganizationReader role in the master
	// account.
	svc := iam.New(sess)
	role, err := awsiam.GetRole(svc, roles.OrganizationAdministrator)
	if err != nil {
		log.Fatal(err)
	}
	if err := role.AddPrincipal(svc, &policies.Principal{
		AWS: []string{roles.Arn(aws.StringValue(account.Id), Administrator)},
	}); err != nil {
		log.Fatal(err)
	}
	role, err = awsiam.GetRole(svc, roles.OrganizationReader)
	if err != nil {
		log.Fatal(err)
	}
	if err := role.AddPrincipal(svc, &policies.Principal{
		AWS: []string{aws.StringValue(account.Id)},
	}); err != nil {
		log.Fatal(err)
	}

	// TODO add this account to the principals of the OrganizationReader role
	// TODO add this to the principals OrganizationAdministrator and NetworkAdministrator (plus whatever we add for the audit and deploy accounts)

	// TODO setup Intranet

}
