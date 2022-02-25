package createadminaccount

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
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

func Main() {
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	create := flag.Bool("create", false, "create a new AWS account, if necessary, without confirmation")
	ignoreServiceQuotas := flag.Bool("ignore-service-quotas", false, "ignore the appearance of any service quota being exhausted and continue anyway")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	quality := flag.String("quality", "", "quality for this new AWS account")
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

	// Ensure the account exists.
	ui.Spin("finding the admin account")
	var account *awsorgs.Account
	createdAccount := false
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
			var deadline time.Time
			if *ignoreServiceQuotas {
				deadline = time.Now()
			}
			account, err = awsorgs.EnsureAccount(
				svc,
				awsservicequotas.NewGlobal(sess),
				accounts.Admin,
				accounts.Admin,
				*quality,
				deadline,
			)
			createdAccount = true
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

	// We used to collect this metadata XML interactively. Now if it's there
	// we use it and if it's not we move along because we're not adding new
	// SAML providers.
	var idpName, metadata string
	if b, err := fileutil.ReadFile(SAMLMetadataFilename); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatal(err)
		}
	} else {
		metadata = string(b)

		// Set idpName early, too, since we'll need it to name the SAML
		// provider we're still managing from the before times.
		if strings.Contains(metadata, "google.com") {
			idpName = Google
		} else {
			idpName = Okta
		}

	}

	svc := iam.New(sess, &aws.Config{
		Credentials: stscreds.NewCredentials(sess, roles.Arn(
			aws.StringValue(account.Id),
			roles.OrganizationAccountAccessRole,
		)),
	})

	var saml *awsiam.SAMLProvider
	if metadata != "" {
		ui.Spinf("configuring %s as your organization's identity provider", idpName)
		saml, err = awsiam.EnsureSAMLProvider(svc, idpName, metadata)
		if err != nil {
			log.Fatal(err)
		}
		ui.Stopf("provider %s", saml.Arn)
		//log.Printf("%+v", saml)
	}

	// Pre-create this user so that it may be referenced in policies attached to
	// the Administrator user.  Terraform will attach policies to it later.
	ui.Spin("finding or creating an IAM user for your Credential Factory, so it can get 12-hour credentials")
	user, err := awsiam.EnsureUser(svc, users.CredentialFactory)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(5e9) // TODO wait only just long enough for IAM to become consistent, and probably do it in EnsureUser
	ui.Stopf("user %s", user.UserName)

	// Create the Administrator role, etc. even without all the principals
	// that need to assume that role because the Terraform run needs to assume
	// the Administrator role. Yes, this is a bit of a Catch-22 but it ends up
	// in a really ergonomic steady state, so we deal with the first run
	// complexity.
	if err := ensureAdministrator(sess, svc, account, createdAccount, saml); err != nil {
		log.Fatal(err)
	}

	// Make arrangements for a hosted zone to appear in this account so that
	// the Intranet can configure itself.  It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	if !fileutil.Exists(choices.IntranetDNSDomainNameFilename) {
		svc := sts.New(sess)
		assumedRole, err := awssts.AssumeRole(
			svc,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
			"substrate-create-admin-account",
			3600,
		)
		if err != nil {
			log.Fatal(err)
		}
		consoleSigninURL, err := awssts.ConsoleSigninURL(
			svc,
			assumedRole.Credentials,
			"https://console.aws.amazon.com/route53/home#DomainListing:",
		)
		if err != nil {
			log.Fatal(err)
		}
		ui.OpenURL(consoleSigninURL)
		ui.Print("buy or transfer a domain into this account or create a hosted zone for a subdomain you've delegated from elsewhere")
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

	// Collect the OAuth OIDC client ID (and secret, below) early now, instead.
	// We need a clue as to which IdP we're using for some of the later steps.
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

	// We might not have gotten to detect idpName above but we'll definitely
	// be able to now that we have an OAuth OIDC client ID.
	if strings.HasSuffix(clientId, ".apps.googleusercontent.com") {
		idpName = Google
	} else {
		idpName = Okta
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
	intranetRegionalProxyModule := terraform.IntranetRegionalProxyModule()
	if err := intranetRegionalProxyModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/regional/proxy")); err != nil {
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
		region := choices.DefaultRegion()

		file := terraform.NewFile()
		module := terraform.Module{
			Arguments: map[string]terraform.Value{
				"dns_domain_name": terraform.Q(dnsDomainName),
			},
			Label: terraform.Q("intranet"),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.UsEast1ProviderAlias: terraform.UsEast1ProviderAlias,
			},
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
		providersFile.Push(terraform.UsEast1Provider(
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
			arguments["okta_hostname"] = terraform.Q(oauthoidc.OktaHostnameValueForGoogleIdP)
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
			/*
			   TODO
			   ╷
			   │ Error: Error creating Security Group: InvalidVpcID.NotFound: The vpc ID 'vpc-07d00424f077dfefb' does not exist
			   │ 	status code: 400, request id: 698779f3-d824-4fe9-bf92-8630201d8135
			   │
			   │   with module.intranet.aws_security_group.instance-factory,
			   │   on ../../../../modules/intranet/regional/main.tf line 414, in resource "aws_security_group" "instance-factory":
			   │  414: resource "aws_security_group" "instance-factory" {
			   │
			   ╵
			   ╷
			   │ Error: Error creating Security Group: InvalidVpcID.NotFound: The vpc ID 'vpc-07d00424f077dfefb' does not exist
			   │ 	status code: 400, request id: 759bee77-0936-4fd4-be8e-9843632507e0
			   │
			   │   with module.intranet.aws_security_group.substrate-instance-factory,
			   │   on ../../../../modules/intranet/regional/main.tf line 425, in resource "aws_security_group" "substrate-instance-factory":
			   │  425: resource "aws_security_group" "substrate-instance-factory" { // remove in 2022.05 with release notes about failure if Instance Factory instances have existed for more than six months
			   │
			   ╵
			   main.go:443: exit status 1
			*/
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
					roles.Arn(aws.StringValue(account.Id), roles.Intranet),       // must match intranet/global/main.tf
					roles.Arn(aws.StringValue(account.Id), "substrate-intranet"), // remove in 2022.01
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

	// Recreate the Administrator and Auditor roles. This is a no-op in steady
	// state but on the first run its assume role policy is missing some
	// principals that were just created in the initial Terraform run.
	if err := ensureAdministrator(sess, svc, account, createdAccount, saml); err != nil {
		log.Fatal(err)
	}

	// Google asks GSuite admins to set custom attributes user by user.  Help
	// these poor souls out by at least telling them exactly what value to set.
	if idpName == Google {
		ui.Printf("set the custom AWS.RoleName attribute in Google for every user to the name of the IAM role they're authorized to assume")
	}

	ui.Printf(
		"next, commit substrate.*, modules/intranet/, modules/lambda-function/, modules/substrate/, and root-modules/%s/%s/ to version control, then run substrate-create-account as many times as you need",
		Domain,
		*quality,
	)
	ui.Printf("you should also start using substrate-credentials or <https://%s/credential-factory> to mint short-lived AWS access keys", dnsDomainName)

}

func ensureAdministrator(sess *session.Session, svc *iam.IAM, account *awsorgs.Account, createdAccount bool, saml *awsiam.SAMLProvider) error {

	{
		_, err := awsiam.GetRole(svc, roles.Intranet)
		log.Print(awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity))
	}
	// Decide whether we're going to include principals created during the
	// Terraform run in the assume role policy.
	var bootstrapping bool
	if _, err := awsiam.GetRole(svc, roles.Intranet); awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
		bootstrapping = true
	}

	// Give the IdP and EC2 some entrypoints in the account.
	ui.Spin("finding or creating roles for your IdP and EC2 to assume in this admin account")
	canned, err := admin.CannedAssumeRolePolicyDocuments(sess, bootstrapping)
	if err != nil {
		return err
	}
	assumeRolePolicyDocument := policies.Merge(
		canned.AdminRolePrincipals, // must be at index 0
		policies.AssumeRolePolicyDocument(&policies.Principal{
			AWS: []string{users.Arn(
				aws.StringValue(account.Id),
				users.CredentialFactory,
			)},
			Service: []string{"ec2.amazonaws.com"},
		}),
	)
	if saml != nil {
		assumeRolePolicyDocument = policies.Merge(
			assumeRolePolicyDocument,
			policies.AssumeRolePolicyDocument(&policies.Principal{
				Federated: []string{saml.Arn},
			}),
		)
	}
	//log.Printf("%+v", assumeRolePolicyDocument)
	if _, err := admin.EnsureAdministratorRole(svc, assumeRolePolicyDocument); err != nil {
		return err
	}
	assumeRolePolicyDocument.Statement[0] = canned.AuditorRolePrincipals.Statement[0] // this is why it must be at index 0
	//log.Printf("%+v", assumeRolePolicyDocument)
	if _, err := admin.EnsureAuditorRole(svc, assumeRolePolicyDocument); err != nil {
		return err
	}
	ui.Stop("ok")

	// This must come before the Terraform run because it references the IAM
	// roles created here.
	admin.EnsureAdminRolesAndPolicies(sess, createdAccount)

	return nil
}
