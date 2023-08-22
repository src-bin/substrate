package setup

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
)

const (
	Domain      = "admin"
	Environment = "admin"
)

// intranet configures the Intranet in the Substrate account and returns the
// DNS domain name where it's being served. This is, at present, mostly a crib
// from `substrate create-admin-account`.
func intranet(ctx context.Context, mgmtCfg, substrateCfg *awscfg.Config) (dnsDomainName string, idpName oauthoidc.Provider) {
	substrateAccountId := substrateCfg.MustAccountId(ctx)
	substrateAccount := awsorgs.Must(awsorgs.DescribeAccount(ctx, mgmtCfg, substrateAccountId))

	quality := substrateAccount.Tags[tagging.Quality]
	if quality == "" {
		qualities, err := naming.Qualities()
		ui.Must(err)
		quality = qualities[0]
		if len(qualities) > 1 {
			ui.Printf(
				"found multiple qualities %s; choosing %s for your Substrate account (this is temporary and inconsequential)",
				strings.Join(qualities, ", "),
				quality,
			)
		}
	}

	// TODO SAML?

	// Make arrangements for a hosted zone to appear in this account so that
	// the Intranet can configure itself.  It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	// TODO allow them to just use the API Gateway hostname when we adopt v2.
	if !fileutil.Exists(naming.IntranetDNSDomainNameFilename) {
		creds, err := substrateCfg.Retrieve(ctx)
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
	var err error
	dnsDomainName, err = ui.PromptFile(
		naming.IntranetDNSDomainNameFilename,
		"what DNS domain name (the one you just bought, transferred, or shared) will you use for your organization's Intranet?",
	)
	ui.Must(err)
	ui.Printf("using DNS domain name %s for your organization's Intranet", dnsDomainName)
	ui.Spinf("waiting for a hosted zone to appear for %s.", dnsDomainName)
	for {
		zone, err := awsroute53.FindHostedZone(ctx, substrateCfg, dnsDomainName+".")
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

	idpName = oauthoidc.IdPName(clientId)
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
		Quality:     quality,
	}
	{
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, quality, regions.Global)
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
			roles.ARN(substrateAccountId, roles.Administrator),
		))
		providersFile.Add(terraform.UsEast1Provider(
			roles.ARN(substrateAccountId, roles.Administrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

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
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, quality, region)

		networkFile := terraform.NewFile()
		dependsOn := networks.ShareVPC(networkFile, substrateAccount, Domain, Environment, quality, region)
		ui.Must(networkFile.Write(filepath.Join(dirname, "network.tf")))

		file := terraform.NewFile()
		arguments := map[string]terraform.Value{
			"dns_domain_name":                    terraform.Q(dnsDomainName),
			"oauth_oidc_client_id":               terraform.Q(clientId),
			"oauth_oidc_client_secret_timestamp": terraform.Q(clientSecretTimestamp),
			"prefix":                             terraform.Q(naming.Prefix()),
			"selected_regions":                   terraform.QSlice(regions.Selected()),
			"stage_name":                         terraform.Q(quality),
			"telemetry":                          terraform.Bool(telemetry.Enabled()),
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
			roles.ARN(substrateAccountId, roles.Administrator),
		))
		networkAccount, err := mgmtCfg.FindSpecialAccount(ctx, accounts.Network)
		ui.Must(err)
		providersFile.Add(terraform.NetworkProviderFor(
			region,
			roles.ARN(aws.ToString(networkAccount.Id), roles.NetworkAdministrator), // TODO a role that only allows sharing VPCs would be a nice safety measure here
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

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
				awscfg.Must(mgmtCfg.AssumeRole(ctx, substrateAccountId, roles.Administrator, time.Hour)).Regional(region),
				fmt.Sprintf("%s-%s", oauthoidc.OAuthOIDCClientSecret, clientId),
				awssecretsmanager.Policy(&policies.Principal{AWS: []string{
					roles.ARN(substrateAccountId, roles.Intranet), // must match intranet/global/main.tf
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

	return
}
