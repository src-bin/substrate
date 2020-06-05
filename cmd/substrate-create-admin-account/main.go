package main

import (
	"flag"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
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

const (
	Domain               = "admin"
	Environment          = "admin"
	OktaMetadataFilename = "substrate.okta.xml"
)

func main() {
	oktaMetadataPathname := flag.String("okta-metadata", OktaMetadataFilename, "pathname of a file containing your Okta SAML provider metadata")
	quality := flag.String("quality", "", "quality for this new AWS account")
	flag.Parse()
	if *quality == "" {
		ui.Fatal(`-quality"..." is required`)
	}
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	if !veqpDoc.Valid(Environment, *quality) {
		ui.Fatalf(`-quality"%s" is not a valid quality for an admin account in your organization`, *quality)
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

	admin.EnsureAdministratorRolesAndPolicies(sess)

	// Write (or rewrite) Terraform resources that create the Intranet in this
	// admin account.
	dirname := fmt.Sprintf("%s-%s-account", Domain, *quality)
	intranet := terraform.NewBlocks()
	for _, region := range regions.Selected() {
		tags := terraform.Tags{
			Domain:      Domain,
			Environment: Environment,
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
	if err := intranet.Write(path.Join(dirname, "intranet.tf")); err != nil {
		log.Fatal(err)
	}

	// Write (or rewrite) some Terraform providers to make everything usable.
	providers := terraform.Provider{
		AccountId:   aws.StringValue(account.Id),
		RoleName:    roles.Administrator,
		SessionName: "Terraform",
	}.AllRegions()
	if err := providers.Write(path.Join(dirname, "providers.tf")); err != nil {
		log.Fatal(err)
	}

	// TODO put the Okta client secret and Okta client secret timestamp into AWS Secrets Manager

	// TODO make -C intranet && terraform init && terraform apply

}
