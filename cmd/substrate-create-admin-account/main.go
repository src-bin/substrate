package main

import (
	"flag"
	"log"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const OktaMetadataFilename = "substrate.okta.xml"

func main() {
	oktaMetadataPathname := flag.String("okta-metadata", OktaMetadataFilename, "pathname of a file containing your Okta SAML provider metadata")
	quality := flag.String("quality", "", "Quality for this new AWS account")
	flag.Parse()
	if *quality == "" {
		ui.Fatal(`-quality"..." is required`)
	}
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	if !veqpDoc.Valid("admin", *quality) {
		ui.Fatalf(`-quality"%s" is not a valid Quality for an admin account in your organization`, *quality)
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

	sess := awssessions.Must(awssessions.InMasterAccount(roles.OrganizationAdministrator, awssessions.Config{}))

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

	// Also allow the Administrator role in this account to assume the
	// NetworkAdministrator role in the network account.
	svc = iam.New(awssessions.Must(awssessions.InSpecialAccount(
		accounts.Network,
		roles.NetworkAdministrator,
		awssessions.Config{},
	)))
	role, err = awsiam.GetRole(svc, roles.NetworkAdministrator)
	if err != nil {
		log.Fatal(err)
	}
	if err := role.AddPrincipal(svc, &policies.Principal{
		AWS: []string{roles.Arn(aws.StringValue(account.Id), Administrator)},
	}); err != nil {
		log.Fatal(err)
	}

	// TODO setup Intranet
	intranet := terraform.NewBlocks()
	for _, region := range regions.Selected() {
		tags := terraform.Tags{
			Domain:      "admin",
			Environment: "admin",
			Quality:     *quality,
			Region:      region,
		}
		intranet.Push(terraform.Module{
			Arguments: map[string]terraform.Value{
				"okta_client_id":               terraform.Q("0oacg1iawaojz8rOo4x6"), // XXX
				"okta_client_secret_timestamp": terraform.Q("2020-05-28T13:19:31Z"), // XXX
				"okta_hostname":                terraform.Q("dev-662445.okta.com"),  // XXX
				"stage_name":                   terraform.Q(*quality),
			},
			Label:    terraform.Label(tags),
			Provider: terraform.ProviderAliasFor(region),
			Source:   terraform.Q("../intranet"),
		})
	}
	if err := intranet.Write(path.Join("admin", *quality, "intranet.tf")); err != nil {
		log.Fatal(err)
	}
	providers := terraform.Provider{
		AccountId:   aws.StringValue(account.Id),
		RoleName:    roles.Administrator,
		SessionName: "Terraform",
	}.AllRegions()
	if err := providers.Write(path.Join("admin", *quality, "providers.tf")); err != nil {
		log.Fatal(err)
	}

	// TODO instruct them to put the Okta client secret and Okta client secret timestamp into AWS Secrets Manager

}
