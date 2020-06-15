package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const (
	Domain                                 = "admin"
	Environment                            = "admin"
	IntranetDNSDomainNameFile              = "substrate.intranet-dns-domain-name"
	OAuthOIDCClientIdFilename              = "substrate.oauth-oidc-client-id"
	OAuthOIDCClientSecretTimestampFilename = "substrate.oauth-oidc-client-secret-timestamp"
	OktaHostnameFilename                   = "substrate.okta-hostname"
	SAMLMetadataFilename                   = "substrate.saml-metadata.xml"
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

	sess, err := awssessions.InMasterAccount(roles.OrganizationAdministrator, awssessions.Config{})
	if err != nil {
		ui.Print("unable to assume the OrganizationAdministrator role, which means this is probably your first time creating an admin account; please provide an access key from your master AWS account")
		accessKeyId, secretAccessKey := awsutil.ReadAccessKeyFromStdin()
		ui.Printf("using access key %s", accessKeyId)
		sess, err = awssessions.InMasterAccount(
			roles.OrganizationAdministrator,
			awssessions.Config{
				AccessKeyId:     accessKeyId,
				SecretAccessKey: secretAccessKey,
			},
		)
	}
	if err != nil {
		log.Fatal(err)
	}

	lines, err := ui.EditFile(
		SAMLMetadataFilename,
		"here is your current identity provider metadata XML:",
		"paste the XML metadata from your identity provider",
	)
	if err != nil {
		log.Fatal(err)
	}
	metadata := strings.Join(lines, "\n") + "\n" // ui.EditFile is line-oriented but this instance isn't

	// Ensure the account exists.
	ui.Spin("finding or creating the admin account")
	account, err := awsorgs.EnsureAccount(organizations.New(sess), accounts.Admin, accounts.Admin, *quality)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	idpName := idp(sess, account, metadata)

	admin.EnsureAdministratorRolesAndPolicies(sess)

	// Make arrangements for a hosted zone to appear in this account so that
	// the intranet can configure itself.  It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	if true { // TODO if IntranetDNSDomainNameFile doesn't exist
		ui.Print("visit <https://console.aws.amazon.com/route53/home#DomainListing:> and buy or transfer a domain into this account")
		ui.Print("or visit <https://console.aws.amazon.com/route53/home#hosted-zones:> and create a hosted zone you've delegated from elsewhere")
		ui.Prompt("when you've finished, press <enter> to continue")
	}
	dnsDomainName, err := ui.PromptFile(
		IntranetDNSDomainNameFile,
		"what DNS domain name (the one you just bought, transferred, or shared) will you use for your organization's intranet?",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using DNS domain name %s for your organization's intranet", dnsDomainName)
	ui.Spinf("waiting for a hosted zone to appear for %s.", dnsDomainName)
	for {
		zone, err := awsroute53.FindHostedZone(
			route53.New(
				awssessions.AssumeRole(sess, aws.StringValue(account.Id), Administrator),
				&aws.Config{},
			),
			dnsDomainName+".",
		)
		if _, ok := err.(awsroute53.HostedZoneNotFoundError); ok {
			time.Sleep(1e9) // TODO exponential backoff
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		ui.Stopf("hosted zone %s", zone.Id)
		break
	}

	var hostname string
	if idpName == Okta {
		hostname, err = ui.PromptFile(
			OktaHostnameFilename,
			"paste the hostname of your Okta installation:",
		)
		if err != nil {
			log.Fatal(err)
		}
		ui.Printf("using Okta hostname %s", hostname)
	}

	clientId, err := ui.PromptFile(
		OAuthOIDCClientIdFilename,
		"paste your OAuth OIDC client ID:",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using OAuth OIDC client ID %s", clientId)

	var clientSecret string
	b, _ := fileutil.ReadFile(OAuthOIDCClientSecretTimestampFilename)
	clientSecretTimestamp := strings.Trim(string(b), "\r\n")
	if clientSecretTimestamp == "" {
		clientSecretTimestamp = time.Now().Format(time.RFC3339)
		clientSecret, err = ui.Prompt("paste your OAuth OIDC client secret:")
		if err != nil {
			log.Fatal(err)
		}
	}

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
		arguments := map[string]terraform.Value{
			"apigateway_role_arn":                         terraform.U(module.Ref(), ".apigateway_role_arn"),
			"dns_domain_name":                             terraform.Q(dnsDomainName),
			"oauth_oidc_client_id":                        terraform.Q(clientId),
			"oauth_oidc_client_secret_timestamp":          terraform.Q(clientSecretTimestamp),
			"selected_regions":                            terraform.QSlice(regions.Selected()),
			"stage_name":                                  terraform.Q(*quality),
			"substrate_credential_factory_role_arn":       terraform.U(module.Ref(), ".substrate_credential_factory_role_arn"),
			"substrate_instance_factory_role_arn":         terraform.U(module.Ref(), ".substrate_instance_factory_role_arn"),
			"substrate_apigateway_authenticator_role_arn": terraform.U(module.Ref(), ".substrate_apigateway_authenticator_role_arn"),
			"substrate_apigateway_authorizer_role_arn":    terraform.U(module.Ref(), ".substrate_apigateway_authorizer_role_arn"),
			"validation_fqdn":                             terraform.U(module.Ref(), ".validation_fqdn"),
		}
		if hostname != "" {
			arguments["okta_hostname"] = terraform.Q(hostname)
		} else {
			arguments["okta_hostname"] = terraform.Q("unused-by-Google-IDP")
		}
		intranet.Push(terraform.Module{
			Arguments: arguments,
			Label:     terraform.Label(tags),
			Provider:  terraform.ProviderAliasFor(region),
			Source:    terraform.Q("../intranet/regional"),
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
	os.Remove("intranet/global/substrate_okta_authenticator.tf") // TODO remove at version 2020.08
	os.Remove("intranet/global/substrate_okta_authorizer.tf")    // TODO remove at version 2020.08
	intranetRegionalModule := terraform.IntranetRegionalModule()
	if err := intranetRegionalModule.Write("intranet/regional"); err != nil {
		log.Fatal(err)
	}
	os.Remove("intranet/regional/substrate_okta_authenticator.tf") // TODO remove at version 2020.08
	os.Remove("intranet/regional/substrate_okta_authorizer.tf")    // TODO remove at version 2020.08
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
	if err := terraform.Root(dirname); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Init(dirname); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Apply(dirname); err != nil {
		log.Fatal(err)
	}

	// Store the OAuth OIDC client secret in AWS Secrets Manager.
	if clientSecret != "" {
		ui.Spin("storing your OAuth OIDC client secret in AWS Secrets Manager")
		for _, region := range regions.Selected() {
			if _, err := awssecretsmanager.EnsureSecret(
				secretsmanager.New(
					awssessions.AssumeRole(sess, aws.StringValue(account.Id), Administrator),
					&aws.Config{Region: aws.String(region)},
				),
				fmt.Sprintf("OAuthOIDCClientSecret-%s", clientId),
				awssecretsmanager.Policy(&policies.Principal{AWS: []string{
					roles.Arn(aws.StringValue(account.Id), "substrate-apigateway-authenticator"), // must match intranet/global/substrate_apigateway_authenticator.tf
					roles.Arn(aws.StringValue(account.Id), "substrate-apigateway-authorizer"),    // must match intranet/global/substrate_apigateway_authorizer.tf
				}}),
				clientSecretTimestamp,
				clientSecret,
			); err != nil {
				log.Fatal(err)
			}
		}
		if err := ioutil.WriteFile(OAuthOIDCClientSecretTimestampFilename, []byte(clientSecretTimestamp+"\n"), 0666); err != nil {
			log.Fatal(err)
		}
		ui.Stop("ok")
		ui.Printf("wrote %s, which you should commit to version control", OAuthOIDCClientSecretTimestampFilename)
	}

	ui.Printf(
		"next, commit %s-%s-account/ to version control, then run substrate-create-account for your various domains",
		Domain,
		*quality,
	)

}
