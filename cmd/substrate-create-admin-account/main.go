package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const (
	Domain                            = "admin"
	Environment                       = "admin"
	IntranetDNSDomainNameFile         = "substrate.intranet-dns-domain-name"
	OktaClientIdFilename              = "substrate.okta-client-id"
	OktaClientSecretTimestampFilename = "substrate.okta-client-secret-timestamp"
	OktaHostnameFilename              = "substrate.okta-hostname"
	OktaMetadataFilename              = "substrate.okta.xml"
)

func main() {
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

	dnsDomainName, err := ui.PromptFile(
		IntranetDNSDomainNameFile,
		"what DNS domain name will you use for your organization's intranet?",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using intranet DNS domain name %s", dnsDomainName)

	lines, err := ui.EditFile(
		OktaMetadataFilename,
		"here is your current identity provider metadata XML:",
		"paste your identity provider metadata XML from Okta",
	)
	if err != nil {
		log.Fatal(err)
	}
	metadata := strings.Join(lines, "\n") + "\n" // ui.EditFile is line-oriented but this instance isn't

	hostname, err := ui.PromptFile(
		OktaHostnameFilename,
		"paste the hostname of your Okta installation:",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using Okta hostname %s", hostname)

	clientId, err := ui.PromptFile(
		OktaClientIdFilename,
		"paste your Okta application's OAuth OIDC client ID:",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using OAuth OIDC client ID %s", clientId)

	var clientSecret string
	b, _ := fileutil.ReadFile(OktaClientSecretTimestampFilename)
	clientSecretTimestamp := strings.Trim(string(b), "\r\n")
	if clientSecretTimestamp == "" {
		clientSecretTimestamp = time.Now().Format(time.RFC3339)
		clientSecret, err = ui.Prompt("paste your Okta application's OAuth OIDC client secret:")
		if err != nil {
			log.Fatal(err)
		}
	}

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
		Arguments: map[string]terraform.Value{
			"dns_domain_name": terraform.Q(dnsDomainName),
		},
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
				"dns_domain_name":                       terraform.Q(dnsDomainName),
				"okta_client_id":                        terraform.Q(clientId),
				"okta_client_secret_timestamp":          terraform.Q(clientSecretTimestamp),
				"okta_hostname":                         terraform.Q(hostname),
				"selected_regions":                      terraform.QSlice(regions.Selected()),
				"stage_name":                            terraform.Q(*quality),
				"substrate_credential_factory_role_arn": terraform.U(module.Ref(), ".substrate_credential_factory_role_arn"),
				"substrate_instance_factory_role_arn":   terraform.U(module.Ref(), ".substrate_instance_factory_role_arn"),
				"substrate_okta_authenticator_role_arn": terraform.U(module.Ref(), ".substrate_okta_authenticator_role_arn"),
				"substrate_okta_authorizer_role_arn":    terraform.U(module.Ref(), ".substrate_okta_authorizer_role_arn"),
				"validation_fqdn":                       terraform.U(module.Ref(), ".validation_fqdn"),
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

	// Store the Okta client secret in AWS Secrets Manager.
	if clientSecret != "" {
		ui.Spin("storing your Okta OAuth OIDC client secret in AWS Secrets Manager")
		for _, region := range regions.Selected() {
			if _, err := awssecretsmanager.EnsureSecret(
				secretsmanager.New(
					awssessions.AssumeRole(sess, aws.StringValue(account.Id), "Administrator"),
					&aws.Config{Region: aws.String(region)},
				),
				fmt.Sprintf("OktaClientSecret-%s", clientId),
				awssecretsmanager.Policy(&policies.Principal{AWS: []string{
					roles.Arn(aws.StringValue(account.Id), "substrate-okta-authenticator"), // must match intranet/global/substrate_okta_authenticator.tf
					roles.Arn(aws.StringValue(account.Id), "substrate-okta-authorizer"),    // must match intranet/global/substrate_okta_authorizer.tf
				}}),
				clientSecretTimestamp,
				clientSecret,
			); err != nil {
				log.Fatal(err)
			}
		}
		if err := ioutil.WriteFile(OktaClientSecretTimestampFilename, []byte(clientSecretTimestamp+"\n"), 0666); err != nil {
			log.Fatal(err)
		}
		ui.Stop("ok")
	}

}
