package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
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
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
)

const (
	Domain                                 = "admin"
	Environment                            = "admin"
	OAuthOIDCClientIdFilename              = "substrate.oauth-oidc-client-id"
	OAuthOIDCClientSecretTimestampFilename = "substrate.oauth-oidc-client-secret-timestamp"
	OktaHostnameFilename                   = "substrate.okta-hostname"
	SAMLMetadataFilename                   = "substrate.saml-metadata.xml"
)

func main() {
	quality := flag.String("quality", "", "quality for this new AWS account")
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	flag.Parse()
	version.Flag()
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
	svc := organizations.New(sess)
	account, err := awsorgs.EnsureAccount(svc, accounts.Admin, accounts.Admin, *quality)
	if err != nil {
		log.Fatal(err)
	}
	if err := accounts.CheatSheet(svc); err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	idpName := idp(sess, account, metadata)

	admin.EnsureAdminRolesAndPolicies(sess)

	// Make arrangements for a hosted zone to appear in this account so that
	// the intranet can configure itself.  It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	if !fileutil.Exists(choices.IntranetDNSDomainNameFilename) {
		ui.Print("visit <https://console.aws.amazon.com/route53/home#DomainListing:> and buy or transfer a domain into this account")
		ui.Print("or visit <https://console.aws.amazon.com/route53/home#hosted-zones:> and create a hosted zone you've delegated from elsewhere")
		ui.Prompt("when you've finished, press <enter> to continue")
	}
	dnsDomainName, err := ui.PromptFile(
		choices.IntranetDNSDomainNameFilename,
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
				awssessions.AssumeRole(sess, aws.StringValue(account.Id), roles.Administrator),
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
	if clientSecret != "" {
		ui.Spin("storing your OAuth OIDC client secret in AWS Secrets Manager")
		for _, region := range regions.Selected() {
			if _, err := awssecretsmanager.EnsureSecret(
				secretsmanager.New(
					awssessions.AssumeRole(sess, aws.StringValue(account.Id), roles.Administrator),
					&aws.Config{Region: aws.String(region)},
				),
				fmt.Sprintf("%s-%s", oauthoidc.OAuthOIDCClientSecret, clientId),
				awssecretsmanager.Policy(&policies.Principal{AWS: []string{
					roles.Arn(aws.StringValue(account.Id), "substrate-apigateway-authenticator"), // must match intranet/global/main.tf
					roles.Arn(aws.StringValue(account.Id), "substrate-apigateway-authorizer"),    // must match intranet/global/main.tf
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

	// Copy module dependencies that are embedded in this binary into the
	// user's source tree.
	intranetGlobalModule := terraform.IntranetGlobalModule()
	if err := intranetGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/global")); err != nil {
		log.Fatal(err)
	}
	intranetRegionalModule := terraform.IntranetRegionalModule()
	if err := intranetRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/regional")); err != nil {
		log.Fatal(err)
	}
	lambdaFunctionGlobalModule := terraform.LambdaFunctionGlobalModule()
	if err := lambdaFunctionGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "lambda-function/global")); err != nil {
		log.Fatal(err)
	}
	lambdaFunctionRegionalModule := terraform.LambdaFunctionRegionalModule()
	if err := lambdaFunctionRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "lambda-function/regional")); err != nil {
		log.Fatal(err)
	}
	substrateGlobalModule := terraform.SubstrateGlobalModule()
	if err := substrateGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "substrate/global")); err != nil {
		log.Fatal(err)
	}
	substrateRegionalModule := terraform.SubstrateRegionalModule()
	if err := substrateRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "substrate/regional")); err != nil {
		log.Fatal(err)
	}

	// Leave the user a place to put their own Terraform code that can be
	// shared between admin accounts of different qualities.
	/*
		if err := terraform.Scaffold(Domain, dirname); err != nil {
			log.Fatal(err)
		}
	*/

	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect every AWS region in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each region")
	}
	tags := terraform.Tags{
		Domain:      Domain,
		Environment: Environment,
		Quality:     *quality,
	}
	{
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, *quality, "global")
		region := "us-east-1"

		file := terraform.NewFile()
		module := terraform.Module{
			Arguments: map[string]terraform.Value{
				"dns_domain_name": terraform.Q(dnsDomainName),
			},
			Label:  terraform.Q("intranet"),
			Source: terraform.Q("../../../../modules/intranet/global"),
		}
		file.Push(module)
		if err := file.Write(filepath.Join(dirname, "main.tf")); err != nil {
			log.Fatal(err)
		}

		outputsFile := terraform.NewFile()
		outputsFile.Push(terraform.Output{
			Label: terraform.Q("validation_fqdn"), // because there is no aws_route53_record data source
			Value: terraform.U(module.Ref(), ".validation_fqdn"),
		})
		if err := outputsFile.Write(filepath.Join(dirname, "outputs.tf")); err != nil {
			log.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Push(terraform.ProviderFor(
			region,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
		))
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Root(dirname, region); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, *quality, region)

		file := terraform.NewFile()
		deployAccount, err := awsorgs.FindSpecialAccount(
			organizations.New(awssessions.Must(awssessions.AssumeRoleMaster(
				sess,
				roles.OrganizationReader,
			))),
			accounts.Deploy,
		)
		if err != nil {
			log.Fatal(err)
		}
		file.Push(terraform.RemoteState{
			Config: terraform.RemoteStateConfig{
				Bucket:        terraform.S3BucketName("us-east-1"),
				DynamoDBTable: terraform.DynamoDBTableName,
				Key:           filepath.Join(terraform.RootModulesDirname, Domain, *quality, "global/terraform.tfstate"),
				Region:        "us-east-1",
				RoleArn:       roles.Arn(aws.StringValue(deployAccount.Id), roles.TerraformStateManager),
			},
			Label: terraform.Q("global"),
		})
		arguments := map[string]terraform.Value{
			"dns_domain_name":                    terraform.Q(dnsDomainName),
			"oauth_oidc_client_id":               terraform.Q(clientId),
			"oauth_oidc_client_secret_timestamp": terraform.Q(clientSecretTimestamp),
			"selected_regions":                   terraform.QSlice(regions.Selected()),
			"stage_name":                         terraform.Q(*quality),
			"validation_fqdn":                    terraform.U("data.terraform_remote_state.global.outputs.validation_fqdn"),
		}
		if hostname != "" {
			arguments["okta_hostname"] = terraform.Q(hostname)
		} else {
			arguments["okta_hostname"] = terraform.Q(oauthoidc.OktaHostnameValueForGoogleIDP)
		}
		tags.Region = region
		file.Push(terraform.Module{
			Arguments: arguments,
			Label:     terraform.Q("intranet"),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.NetworkProviderAlias: terraform.NetworkProviderAlias,
			},
			Source: terraform.Q("../../../../modules/intranet/regional"),
		})
		if err := file.Write(filepath.Join(dirname, "main.tf")); err != nil {
			log.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Push(terraform.ProviderFor(
			region,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
		))
		networkAccount, err := awsorgs.FindSpecialAccount(organizations.New(awssessions.Must(awssessions.InMasterAccount(
			roles.OrganizationReader,
			awssessions.Config{},
		))), accounts.Network)
		if err != nil {
			log.Fatal(err)
		}
		providersFile.Push(terraform.NetworkProviderFor(
			region,
			roles.Arn(aws.StringValue(networkAccount.Id), roles.Auditor),
		))
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Root(dirname, region); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Printf(
		"next, commit modules/intranet/, modules/lambda-function/, modules/substrate/, and root-modules/%s/%s/ to version control, then run substrate-create-account as many times as you need",
		Domain,
		*quality,
	)
	ui.Printf("you should also start using <https://%s/credential-factory> to mint short-lived AWS access keys", dnsDomainName)

}
