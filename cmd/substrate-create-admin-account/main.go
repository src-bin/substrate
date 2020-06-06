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

	dirname := fmt.Sprintf("%s-%s-account", Domain, *quality)

	// Write (or rewrite) Terraform resources that create the Intranet in this
	// admin account.
	intranet := terraform.NewFile()
	tags := terraform.Tags{
		Domain:      Domain,
		Environment: Environment,
		Quality:     *quality,
	}
	module := terraform.Module{
		Label:    terraform.Label(tags),
		Provider: terraform.GlobalProviderAlias(),
		Source:   terraform.Q("../intranet/global"),
	}
	intranet.Push(module)
	for _, region := range regions.Selected() {
		tags.Region = region
		intranet.Push(terraform.Module{
			Arguments: map[string]terraform.Value{
				"apigateway_role_arn":                   terraform.U(module.Ref(), ".apigateway_role_arn"),
				"okta_client_id":                        terraform.Q("0oacg1iawaojz8rOo4x6"), // XXX
				"okta_client_secret_timestamp":          terraform.Q("2020-05-28T13:19:31Z"), // XXX
				"okta_hostname":                         terraform.Q("dev-662445.okta.com"),  // XXX
				"stage_name":                            terraform.Q(*quality),
				"substrate_instance_factory_role_arn":   terraform.U(module.Ref(), ".substrate_instance_factory_role_arn"),
				"substrate_okta_authenticator_role_arn": terraform.U(module.Ref(), ".substrate_okta_authenticator_role_arn"),
				"substrate_okta_authorizer_role_arn":    terraform.U(module.Ref(), ".substrate_okta_authorizer_role_arn"),
			},
			Label:    terraform.Label(tags),
			Provider: terraform.ProviderAliasFor(region),
			Source:   terraform.Q("../intranet/regional"),
		})
	}
	if err := intranet.Write(path.Join(dirname, "intranet.tf")); err != nil {
		log.Fatal(err)
	}

	// Write (or rewrite) the Terraform modules we referenced (even indirectly)
	// just above.
	intranetGlobalModule := terraform.IntranetGlobalModule()
	if err := intranetGlobalModule.Write("intranet/global"); err != nil {
		log.Fatal(err)
	}
	intranetRegionalModule := terraform.IntranetRegionalModule()
	if err := intranetRegionalModule.Write("intranet/regional"); err != nil {
		log.Fatal(err)
	}
	lambdaFunctionGlobalModule := terraform.LambdaFunctionGlobalModule()
	if err := lambdaFunctionGlobalModule.Write("lambda-function/global"); err != nil {
		log.Fatal(err)
	}
	lambdaFunctionRegionalModule := terraform.LambdaFunctionRegionalModule()
	if err := lambdaFunctionRegionalModule.Write("lambda-function/regional"); err != nil {
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

	// TODO make -C intranet

	// Generate a Makefile in the root Terraform module then apply the generated
	// Terraform code.
	if err := terraform.Makefile(dirname); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Init(dirname); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Apply(dirname); err != nil {
		log.Fatal(err)
	}

}
