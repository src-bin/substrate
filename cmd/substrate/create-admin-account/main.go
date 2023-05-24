package createadminaccount

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

const (
	Domain      = "admin"
	Environment = "admin"

	AzureADTenantFilename = "substrate.azure-ad-tenant"
	OktaHostnameFilename  = "substrate.okta-hostname"
	SAMLMetadataFilename  = "substrate.saml-metadata.xml"

	OAuthOIDCClientIdFilename              = "substrate.oauth-oidc-client-id"
	OAuthOIDCClientSecretTimestampFilename = "substrate.oauth-oidc-client-secret-timestamp"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	create := flag.Bool("create", false, "create a new AWS account, if necessary, without confirmation")
	ignoreServiceQuotas := flag.Bool("ignore-service-quotas", false, "ignore the appearance of any service quota being exhausted and continue anyway")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	quality := cmdutil.QualityFlag("quality for this new AWS account")
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate create-admin-account [-create] [-quality <quality>] [-auto-approve|-no-apply] [-ignore-service-quotas]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *quality == "" {
		ui.Fatal(`-quality "..." is required`)
	}
	veqpDoc, err := veqp.ReadDocument()
	ui.Must(err)
	if !veqpDoc.Valid(Environment, *quality) {
		ui.Fatalf(`-quality %q is not a valid quality for an admin account in your organization`, *quality)
	}

	if _, err = cfg.GetCallerIdentity(ctx); err != nil {
		_, err = cfg.SetRootCredentials(ctx)
		ui.Must(err)
	}
	cfg = awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.OrganizationAdministrator,
		time.Hour,
	))
	versionutil.PreventDowngrade(ctx, cfg)

	// Ensure the account exists.
	ui.Spin("finding the admin account")
	var account *awsorgs.Account
	createdAccount := false
	{
		account, err = cfg.FindAdminAccount(ctx, *quality)
		ui.Must(err)
		if account == nil {
			ui.Stop("not found")
			if !*create {
				if ok, err := ui.Confirmf("create a new %s-quality admin account? (yes/no)", *quality); err != nil {
					ui.Fatal(err)
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
				ctx,
				cfg,
				accounts.Admin,
				accounts.Admin,
				*quality,
				deadline,
			)
			createdAccount = true
		} else {
			err = awsorgs.Tag(
				ctx,
				cfg,
				aws.ToString(account.Id),
				tagging.Map{tagging.SubstrateVersion: version.Version},
			)
		}
		ui.Must(err)
		ui.Must(accounts.CheatSheet(ctx, awscfg.Must(cfg.OrganizationReader(ctx))))
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	cfg.Telemetry().FinalAccountId = aws.ToString(account.Id)
	cfg.Telemetry().FinalRoleName = roles.Administrator

	// We used to collect this metadata XML interactively. Now if it's there
	// we use it and if it's not we move along because we're not adding new
	// SAML providers.
	var metadata string
	if b, err := fileutil.ReadFile(SAMLMetadataFilename); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			ui.Fatal(err)
		}
	} else {
		metadata = string(b)
	}

	adminCfg := awscfg.Must(cfg.AssumeRole(
		ctx,
		aws.ToString(account.Id),
		roles.OrganizationAccountAccessRole,
		time.Hour,
	))

	var saml *awsiam.SAMLProvider
	if metadata != "" {

		// At the last time Substrate helped folks configure a SAML provider,
		// it only supported Google and Okta. Thus, though it supports more
		// IdPs now, we don't have to accommodate any more of them here.
		var idpName oauthoidc.Provider
		if strings.Contains(metadata, "google.com") {
			idpName = oauthoidc.Google
		} else {
			idpName = oauthoidc.Okta
		}

		ui.Spinf("configuring %s as your organization's SAML identity provider (a legacy of very early Substrate releases)", idpName)
		saml, err = awsiam.EnsureSAMLProvider(ctx, adminCfg, idpName, metadata)
		if err != nil {
			ui.Fatal(err)
		}
		ui.Stopf("provider %s", saml.Arn)
		//log.Printf("%+v", saml)
	}

	// Pre-create this user so that it may be referenced in policies attached to
	// the Administrator user.  Terraform will attach policies to it later.
	ui.Spin("finding or creating an IAM user for your Credential Factory, so it can get 12-hour credentials")
	user, err := awsiam.EnsureUser(ctx, adminCfg, users.CredentialFactory)
	ui.Must(err)
	time.Sleep(5e9) // TODO wait only just long enough for IAM to become consistent, and probably do it in EnsureUser
	ui.Stopf("user %s", user.UserName)

	// Create the Administrator role, etc. even without all the principals
	// that need to assume that role because the Terraform run needs to assume
	// the Administrator role. Yes, this is a bit of a Catch-22 but it ends up
	// in a really ergonomic steady state, so we deal with the first run
	// complexity.
	ui.Must(ensureAdministrator(ctx, cfg, adminCfg, account, createdAccount, saml))

	// TODO create Terraformer here and probably don't have to create Administrator yet

	// Make arrangements for a hosted zone to appear in this account so that
	// the Intranet can configure itself.  It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	if !fileutil.Exists(naming.IntranetDNSDomainNameFilename) {
		creds, err := awscfg.Must(cfg.AssumeRole(
			ctx,
			aws.ToString(account.Id),
			roles.Administrator, // this is why we can't reuse adminCfg
			time.Hour,
		)).Retrieve(ctx)
		ui.Must(err)
		consoleSigninURL, err := federation.ConsoleSigninURL(
			creds,
			"https://console.aws.amazon.com/route53/home#DomainListing:", // destination
			nil,
		)
		ui.Must(err)
		ui.OpenURL(consoleSigninURL)
		ui.Print("buy or transfer a domain into this account or create a hosted zone for a subdomain you've delegated from elsewhere")
		ui.Prompt("when you've finished, press <enter> to continue")
	}
	dnsDomainName, err := ui.PromptFile(
		naming.IntranetDNSDomainNameFilename,
		"what DNS domain name (the one you just bought, transferred, or shared) will you use for your organization's Intranet?",
	)
	ui.Must(err)
	ui.Printf("using DNS domain name %s for your organization's Intranet", dnsDomainName)
	ui.Spinf("waiting for a hosted zone to appear for %s.", dnsDomainName)
	for {
		zone, err := awsroute53.FindHostedZone(ctx, adminCfg, dnsDomainName+".")
		if _, ok := err.(awsroute53.HostedZoneNotFoundError); ok {
			time.Sleep(1e9) // TODO exponential backoff
			continue
		}
		ui.Must(err)
		ui.Stopf("hosted zone %s", zone.Id)
		break
	}

	// Collect the OAuth OIDC client ID (and secret, below) early now, instead.
	// We need a clue as to which IdP we're using for some of the later steps.
	clientId, err := ui.PromptFile(
		OAuthOIDCClientIdFilename,
		"paste your OAuth OIDC client ID:",
	)
	ui.Must(err)
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
		ui.Must(err)
	}

	idpName := oauthoidc.IdPName(clientId)
	ui.Printf("configuring %s as your organization's OAuth OIDC identity provider", idpName)

	var hostname, tenantId string
	switch idpName {
	case oauthoidc.AzureAD:
		tenantId, err = ui.PromptFile(
			AzureADTenantFilename,
			"paste the tenant ID from your Azure AD installation:",
		)
		ui.Must(err)
		ui.Printf("using Azure AD tenant ID %s", tenantId)
	case oauthoidc.Okta:
		hostname, err = ui.PromptFile(
			OktaHostnameFilename,
			"paste the hostname of your Okta installation:",
		)
		ui.Must(err)
		ui.Printf("using Okta hostname %s", hostname)
	}

	// Copy module dependencies that are embedded in this binary into the
	// user's source tree.
	intranetGlobalModule := terraform.IntranetGlobalModule()
	ui.Must(intranetGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/global")))
	intranetRegionalModule := terraform.IntranetRegionalModule()
	ui.Must(intranetRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/regional")))
	intranetRegionalProxyModule := terraform.IntranetRegionalProxyModule()
	ui.Must(intranetRegionalProxyModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/regional/proxy")))
	lambdaFunctionGlobalModule := terraform.LambdaFunctionGlobalModule()
	ui.Must(lambdaFunctionGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "lambda-function/global")))
	lambdaFunctionRegionalModule := terraform.LambdaFunctionRegionalModule()
	ui.Must(lambdaFunctionRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "lambda-function/regional")))
	substrateGlobalModule := terraform.SubstrateGlobalModule()
	ui.Must(substrateGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "substrate/global")))
	substrateRegionalModule := terraform.SubstrateRegionalModule()
	ui.Must(substrateRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "substrate/regional")))

	// Leave the user a place to put their own Terraform code that can be
	// shared between admin accounts of different qualities.
	/*
		ui.Must(terraform.Scaffold(Domain, true))
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
		region := regions.Default()

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
		file.Add(module)
		ui.Must(file.Write(filepath.Join(dirname, "main.tf")))

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
		))
		providersFile.Add(terraform.UsEast1Provider(
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, cfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))

		ui.Must(terraform.Init(dirname))

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		ui.Must(err)
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, *quality, region)

		networkFile := terraform.NewFile()
		dependsOn := networks.ShareVPC(networkFile, account, Domain, Environment, *quality, region)
		ui.Must(networkFile.Write(filepath.Join(dirname, "network.tf")))

		file := terraform.NewFile()
		arguments := map[string]terraform.Value{
			"dns_domain_name":                    terraform.Q(dnsDomainName),
			"oauth_oidc_client_id":               terraform.Q(clientId),
			"oauth_oidc_client_secret_timestamp": terraform.Q(clientSecretTimestamp),
			"prefix":                             terraform.Q(naming.Prefix()),
			"selected_regions":                   terraform.QSlice(regions.Selected()),
			"stage_name":                         terraform.Q(*quality),
		}
		if hostname != "" {
			arguments["okta_hostname"] = terraform.Q(hostname)
		} else {
			arguments["okta_hostname"] = terraform.Q(oauthoidc.OktaHostnameValueForNonOktaIdP)
		}
		if tenantId != "" {
			arguments["azure_ad_tenant_id"] = terraform.Q(tenantId)
		} else {
			arguments["azure_ad_tenant_id"] = terraform.Q(oauthoidc.AzureADTenantValueForNonAzureADIdP)
		}
		tags.Region = region
		file.Add(terraform.Module{
			Arguments: arguments,
			DependsOn: dependsOn,
			Label:     terraform.Q("intranet"),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.NetworkProviderAlias: terraform.NetworkProviderAlias,
			},
			Source: terraform.Q("../../../../modules/intranet/regional"),
		})
		ui.Must(file.Write(filepath.Join(dirname, "main.tf")))

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
		))
		networkAccount, err := cfg.FindSpecialAccount(ctx, accounts.Network)
		ui.Must(err)
		providersFile.Add(terraform.NetworkProviderFor(
			region,
			roles.ARN(aws.ToString(networkAccount.Id), roles.NetworkAdministrator), // TODO a role that only allows sharing VPCs would be a nice safety measure here
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, cfg, dirname, region))

		ui.Must(terraform.Init(dirname))

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		ui.Must(err)
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	// Now, after the (first) Terraform run, we'll be able to set the necessary
	// policy on the client secret in AWS Secrets Manager.
	if clientSecret != "" {
		ui.Spin("storing your OAuth OIDC client secret in AWS Secrets Manager")
		for _, region := range regions.Selected() {
			_, err := awssecretsmanager.EnsureSecret(
				ctx,
				awscfg.Must(cfg.AssumeRole(
					ctx,
					aws.ToString(account.Id),
					roles.Administrator,
					time.Hour,
				)).Regional(region),
				fmt.Sprintf("%s-%s", oauthoidc.OAuthOIDCClientSecret, clientId),
				awssecretsmanager.Policy(&policies.Principal{AWS: []string{
					roles.ARN(aws.ToString(account.Id), roles.Intranet), // must match intranet/global/main.tf
				}}),
				clientSecretTimestamp,
				clientSecret,
			)
			ui.Must(err)
		}
		ui.Must(ioutil.WriteFile(OAuthOIDCClientSecretTimestampFilename, []byte(clientSecretTimestamp+"\n"), 0666))
		ui.Stop("ok")
		ui.Printf("wrote %s, which you should commit to version control", OAuthOIDCClientSecretTimestampFilename)
	}

	// Recreate the Administrator and Auditor roles. This is a no-op in steady
	// state but on the first run its assume role policy is missing some
	// principals that were just created in the initial Terraform run.
	ui.Must(ensureAdministrator(ctx, cfg, adminCfg, account, createdAccount, saml))

	// Use API Gateway v2, Lambda, etc. to construct the Intranet. This one's
	// better than the original that was implemented in Terraform because it
	// ends up being a lot easier for folks to extend.
	ui.Must(ensureIntranet(ctx, adminCfg, dnsDomainName, clientId, clientSecret, hostname, tenantId))

	// Google asks GSuite admins to set custom attributes user by user.  Help
	// these poor souls out by at least telling them exactly what value to set.
	if idpName == oauthoidc.AzureAD || idpName == oauthoidc.Google {
		ui.Printf("set the custom AWS.RoleName attribute in %s for every user to the name of the IAM role they're authorized to assume", idpName)
	}

	ui.Print("next, commit the following files to version control:")
	ui.Print("")
	ui.Print("substrate.*")
	//ui.Printf(OAuthOIDCClientSecretTimestampFilename) // covered by substrate.*
	ui.Print("modules/intranet/")
	ui.Print("modules/lambda-function/")
	ui.Print("modules/substrate/")
	ui.Printf("root-modules/%s/%s/", Domain, *quality)
	ui.Print("")
	ui.Print("then, run `substrate create-account` to create the service accounts you need")
	ui.Printf("you should also start using `eval $(substrate credentials)` or <https://%s/credential-factory> to mint short-lived AWS access keys", dnsDomainName)

}

// ensureAdministrator configures the Administrator and Auditor roles in all
// the AWS accounts. It must be called with a cfg in the management account and
// an adminCfg in the admin account being managed by this command.
func ensureAdministrator(
	ctx context.Context,
	cfg, adminCfg *awscfg.Config,
	account *awsorgs.Account,
	createdAccount bool,
	saml *awsiam.SAMLProvider,
) error {

	// Decide whether we're going to include principals created during the
	// Terraform run in the assume role policy.
	var bootstrapping bool
	if _, err := awsiam.GetRole(ctx, adminCfg, roles.Intranet); awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
		bootstrapping = true
	}

	// Give the IdP and EC2 some entrypoints in the account.
	ui.Spin("finding or creating roles for your IdP and EC2 to assume in this admin account")
	canned, err := admin.CannedAssumeRolePolicyDocuments(ctx, cfg, bootstrapping)
	if err != nil {
		return err
	}
	assumeRolePolicyDocument := policies.Merge(
		canned.AdminRolePrincipals, // must be at index 0
		policies.AssumeRolePolicyDocument(&policies.Principal{
			AWS: []string{users.ARN(
				aws.ToString(account.Id),
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
	if _, err := admin.EnsureAdministratorRole(ctx, adminCfg, assumeRolePolicyDocument); err != nil {
		return err
	}
	assumeRolePolicyDocument.Statement[0] = canned.AuditorRolePrincipals.Statement[0] // this is why it must be at index 0
	//log.Printf("%+v", assumeRolePolicyDocument)
	if _, err := admin.EnsureAuditorRole(ctx, adminCfg, assumeRolePolicyDocument); err != nil {
		return err
	}
	ui.Stop("ok")

	// This must come before the Terraform run because it references the IAM
	// roles created here.
	admin.EnsureAdminRolesAndPolicies(ctx, cfg, createdAccount)

	return nil
}
