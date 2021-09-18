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
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
)

const (
	Domain      = "admin"
	Environment = "admin"

	Google = "Google"
	Okta   = "Okta"

	OAuthOIDCClientIdFilename              = "substrate.oauth-oidc-client-id"
	OAuthOIDCClientSecretTimestampFilename = "substrate.oauth-oidc-client-secret-timestamp"
	OktaHostnameFilename                   = "substrate.okta-hostname"

	SAMLMetadataFilename = "substrate.saml-metadata.xml"
)

func main() {
	quality := flag.String("quality", "", "quality for this new AWS account")
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	create := flag.Bool("create", false, "create a new AWS account, if necessary, without confirmation")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	cmdutil.MustChdir()
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
		ui.Fatalf(`-quality="%s" is not a valid quality for an admin account in your organization`, *quality)
	}

	sess, err := awssessions.InManagementAccount(roles.OrganizationAdministrator, awssessions.Config{
		FallbackToRootCredentials: true,
	})
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
	ui.Spin("finding the admin account")
	var account *awsorgs.Account
	{
		svc := organizations.New(sess)
		account, err = awsorgs.FindAccount(svc, accounts.Admin, accounts.Admin, *quality)
		if _, ok := err.(awsorgs.AccountNotFound); ok {
			ui.Stop("not found")
			if !*create {
				if ok, err := ui.Confirmf("create a new %s-quality admin account? (yes/no)", *quality); err != nil {
					log.Fatal(err)
				} else if !ok {
					ui.Fatal("not creating a new AWS account")
				}
			}
			ui.Spin("creating the admin account")
			account, err = awsorgs.EnsureAccount(svc, accounts.Admin, accounts.Admin, *quality)
		}
		if err != nil {
			log.Fatal(err)
		}
		if err := accounts.CheatSheet(svc); err != nil {
			log.Fatal(err)
		}
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	//idpName := idp(sess, account, metadata)
	svc := iam.New(sess, &aws.Config{
		Credentials: stscreds.NewCredentials(sess, roles.Arn(
			aws.StringValue(account.Id),
			roles.OrganizationAccountAccessRole,
		)),
	})

	idpName := "IdP"
	if strings.Contains(metadata, "google.com") {
		idpName = Google
	} else if strings.Contains(metadata, "okta.com") {
		idpName = Okta
	}

	ui.Spinf("configuring %s as your organization's identity provider", idpName)
	saml, err := awsiam.EnsureSAMLProvider(svc, idpName, metadata)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("provider %s", saml.Arn)
	//log.Printf("%+v", saml)

	// Pre-create this user so that it may be referenced in policies attached to
	// the Administrator user.  Terraform will attach policies to it later.
	ui.Spin("finding or creating an IAM user for your Credential Factory, so it can get 12-hour credentials")
	user, err := awsiam.EnsureUser(svc, users.CredentialFactory)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(5e9) // TODO wait only just long enough for IAM to become consistent, and probably do it in EnsureUser
	ui.Stopf("user %s", user.UserName)

	// Give Okta some entrypoints in the Admin account.
	ui.Spinf("finding or creating roles for %s to use in this admin account", idpName)
	_, adminRolePrincipals, err := admin.AdminPrincipals(organizations.New(sess))
	if err != nil {
		log.Fatal(err)
	}
	assumeRolePolicyDocument := &policies.Document{
		Statement: []policies.Statement{
			policies.AssumeRolePolicyDocument(adminRolePrincipals).Statement[0],
			policies.AssumeRolePolicyDocument(&policies.Principal{
				AWS: []string{users.Arn(
					aws.StringValue(account.Id),
					users.CredentialFactory,
				)},
				Service: []string{"ec2.amazonaws.com"},
			}).Statement[0],
			policies.AssumeRolePolicyDocument(&policies.Principal{
				Federated: []string{saml.Arn},
			}).Statement[0],
		},
	}
	//log.Printf("%+v", assumeRolePolicyDocument)
	if _, err := admin.EnsureAdministratorRole(svc, assumeRolePolicyDocument); err != nil {
		log.Fatal(err)
	}
	if _, err := admin.EnsureAuditorRole(svc, assumeRolePolicyDocument); err != nil {
		log.Fatal(err)
	}
	ui.Stop("ok")

	// This must come before the Terraform run because it references the IAM
	// roles created here.
	admin.EnsureAdminRolesAndPolicies(sess)

	// Make arrangements for a hosted zone to appear in this account so that
	// the Intranet can configure itself.  It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	if !fileutil.Exists(choices.IntranetDNSDomainNameFilename) {
		ui.Print("visit <https://console.aws.amazon.com/route53/home#DomainListing:> and buy or transfer a domain into this account")
		ui.Print("or visit <https://console.aws.amazon.com/route53/home#hosted-zones:> and create a hosted zone you've delegated from elsewhere")
		ui.Prompt("when you've finished, press <enter> to continue")
	}
	dnsDomainName, err := ui.PromptFile(
		choices.IntranetDNSDomainNameFilename,
		"what DNS domain name (the one you just bought, transferred, or shared) will you use for your organization's Intranet?",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using DNS domain name %s for your organization's Intranet", dnsDomainName)
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

	// Collect the OAuth OIDC client secret now but don't store it permanently
	// yet. We can't set the access policy it needs in AWS Secrets Manager
	// until the authorized principals exist, which means we must wait until
	// after the (first) Terraform run.
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
	if err := ioutil.WriteFile(
		filepath.Join(terraform.ModulesDirname, "intranet/regional/substrate-intranet.zip"),
		SubstrateIntranetZip,
		0666,
	); err != nil {
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
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, *quality, regions.Global)
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
		arguments := map[string]terraform.Value{
			"dns_domain_name":                    terraform.Q(dnsDomainName),
			"oauth_oidc_client_id":               terraform.Q(clientId),
			"oauth_oidc_client_secret_timestamp": terraform.Q(clientSecretTimestamp),
			"selected_regions":                   terraform.QSlice(regions.Selected()),
			"stage_name":                         terraform.Q(*quality),
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

		networkFile := terraform.NewFile()
		networks.ShareVPC(networkFile, account, Domain, Environment, *quality, region)
		if err := networkFile.Write(filepath.Join(dirname, "network.tf")); err != nil {
			log.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Push(terraform.ProviderFor(
			region,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
		))
		networkAccount, err := awsorgs.FindSpecialAccount(organizations.New(awssessions.Must(awssessions.InManagementAccount(
			roles.OrganizationReader,
			awssessions.Config{},
		))), accounts.Network)
		if err != nil {
			log.Fatal(err)
		}
		providersFile.Push(terraform.NetworkProviderFor(
			region,
			roles.Arn(aws.StringValue(networkAccount.Id), roles.NetworkAdministrator), // TODO a role that only allows sharing VPCs would be a nice safety measure here
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

	// Now, after the (first) Terraform run, we'll be able to set the necessary
	// policy on the client secret in AWS Secrets Manager.
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
					roles.Arn(aws.StringValue(account.Id), "substrate-apigateway-authorizer"), // must match intranet/global/main.tf // TODO remove in 2021.10
					roles.Arn(aws.StringValue(account.Id), "substrate-intranet"),              // must match intranet/global/main.tf
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

	// Google asks GSuite admins to set custom attributes user by user.  Help
	// these poor souls out by at least telling them exactly what value to set.
	if idpName == Google {
		ui.Printf(
			`set the AWS/Role custom attribute in GSuite for every authorized AWS Console user to "%s,%s"`,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
			saml.Arn,
		)
	}

	// Give Okta a user that can enumerate the roles it can assume.  Only Okta
	// needs this.  Google puts more of the burden on GSuite admins.
	if idpName == Okta {
		ui.Spin("finding or creating a user for Okta to use to enumerate roles")
		user, err := awsiam.EnsureUserWithPolicy(
			svc,
			Okta,
			&policies.Document{
				Statement: []policies.Statement{{
					Action:   []string{"iam:ListAccountAliases", "iam:ListRoles"},
					Resource: []string{"*"},
				}},
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		ui.Stopf("user %s", user.UserName)
		//log.Printf("%+v", user)
		var ok bool
		if ui.Interactivity() == ui.FullyInteractive {
			if ok, err = ui.Confirm( // TODO need to find a way to remove this as it is remarkably confusing and tedious if answered incorrectly; maybe only do it if IdP metadata was changed?
				`answering "yes" will break any existing integration - do you need to configure Okta's AWS integration? (yes/no)`,
			); err != nil {
				log.Fatal(err)
			}
		} else {
			ui.Print("if you need to reconfigure Okta, re-run this command with -fully-interactive")
			time.Sleep(5e9)
		}
		if ok {
			ui.Spin("deleting existing access keys and creating a new one")
			if err := awsiam.DeleteAllAccessKeys(svc, Okta); err != nil {
				log.Fatal(err)
			}
			accessKey, err := awsiam.CreateAccessKey(svc, aws.StringValue(user.UserName))
			if err != nil {
				log.Fatal(err)
			}
			ui.Stop("ok")
			//log.Printf("%+v", accessKey)
			ui.Printf("Okta needs this SAML provider ARN: %s", saml.Arn)
			ui.Printf(".. and this access key ID: %s", accessKey.AccessKeyId)
			ui.Printf("...and this secret access key: %s", accessKey.SecretAccessKey) // TODO can't put this into substrate.onboarding.txt; shit
			if _, err := ui.Prompt("press <enter> after you've updated your Okta configuration"); err != nil {
				log.Fatal(err)
			}
		}
	}

	ui.Printf(
		"next, commit modules/intranet/, modules/lambda-function/, modules/substrate/, and root-modules/%s/%s/ to version control, then run substrate-create-account as many times as you need",
		Domain,
		*quality,
	)
	ui.Printf("you should also start using substrate-credentials or <https://%s/credential-factory> to mint short-lived AWS access keys", dnsDomainName)

}
