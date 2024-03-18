package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsapigatewayv2"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscloudfront"
	"github.com/src-bin/substrate/awslambda"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awsutil"
	intranetzip "github.com/src-bin/substrate/cmd/substrate/intranet-zip"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
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

// intranet configures the Intranet in the Substrate account and returns the
// DNS domain name where it's being served. This is entirely deprecated in
// favor of intranet2 and is slowly being dismantled and removed.
func intranet(ctx context.Context, mgmtCfg, substrateCfg *awscfg.Config) (dnsDomainName string, idpName oauthoidc.Provider) {

	// Gather configuration first.

	substrateAccountId := substrateCfg.MustAccountId(ctx)
	networkCfg := awscfg.Must(mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour))
	roleARN := roles.ARN(substrateAccountId, roles.Substrate)

	quality := ui.Must2(awsorgs.Must(awsorgs.DescribeAccount(ctx, mgmtCfg, substrateAccountId)).Quality())

	// Make arrangements for a hosted zone to appear in this account so that
	// the Intranet can configure itself. It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	// TODO allow them to just use the CloudFront hostname.
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
	var zone *awsroute53.HostedZone
	for range awsutil.StandardJitteredExponentialBackoff() {
		zone, err = awsroute53.FindHostedZone(ctx, substrateCfg, dnsDomainName+".")
		if _, ok := err.(awsroute53.HostedZoneNotFoundError); ok {
			continue
		}
		ui.Must(err)
		ui.Stopf("hosted zone %s", zone.Id)
		break
	}

	// Collect the OAuth OIDC client ID and secret. Store the secret in AWS
	// Secrets Manager and refer to it with a timestamp.
	clientId, err := ui.PromptFile(
		OAuthOIDCClientIdFilename,
		"paste your OAuth OIDC client ID:",
	)
	ui.Must(err)
	ui.Printf("using OAuth OIDC client ID %s", clientId)
	b, _ := os.ReadFile(OAuthOIDCClientSecretTimestampFilename)
	clientSecretTimestamp := strings.Trim(string(b), "\r\n")
	if clientSecretTimestamp == "" {
		clientSecretTimestamp = time.Now().Format(time.RFC3339)
		clientSecret, err := ui.Prompt("paste your OAuth OIDC client secret:")
		ui.Must(err)
		ui.Spin("storing your OAuth OIDC client secret in AWS Secrets Manager")
		for _, region := range regions.Selected() {
			_, err := awssecretsmanager.EnsureSecret(
				ctx,
				substrateCfg.Regional(region),
				fmt.Sprintf("%s-%s", oauthoidc.OAuthOIDCClientSecret, clientId),
				awssecretsmanager.Policy(&policies.Principal{AWS: []string{roleARN}}),
				clientSecretTimestamp,
				clientSecret,
			)
			ui.Must(err)
		}
		ui.Must(os.WriteFile(OAuthOIDCClientSecretTimestampFilename, []byte(clientSecretTimestamp+"\n"), 0666))
		ui.Stop("ok")
		ui.Printf("wrote %s, which you should commit to version control", OAuthOIDCClientSecretTimestampFilename)
	}

	// Collect whatever additional information we need, depending on which sort
	// of IdP they're using.
	idpName = oauthoidc.IdPName(clientId)
	ui.Printf("configuring %s as your organization's OAuth OIDC identity provider", idpName)
	hostname := oauthoidc.OktaHostnameValueForNonOktaIdP
	tenantId := oauthoidc.AzureADTenantValueForNonAzureADIdP
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

	// We've finished gathering configuration.
	//
	// Find or create the API Gateway v2-based Intranet.

	// Prepare the Lambda function's environment variables. If we can find our
	// CloudFront distribution, include its DNS domain name in the environment
	// right away to prevent a transient outage while we reconfigure the Lambda
	// function twice, once before managing the CloudFront distribution and
	// once again after. On first runs, the Lambda function will be partially
	// configured before the CloudFront distribution is created and then
	// reconfigured to include the CloudFront distribution's DNS domain name.
	environment := map[string]string{
		"AZURE_AD_TENANT_ID":                 tenantId,
		"OAUTH_OIDC_CLIENT_ID":               clientId,
		"OAUTH_OIDC_CLIENT_SECRET_TIMESTAMP": clientSecretTimestamp,
		"OKTA_HOSTNAME":                      hostname,
		"SELECTED_REGIONS":                   strings.Join(regions.Selected(), ","),
		"SUBSTRATE_PREFIX":                   naming.Prefix(),
	}
	if distribution, err := awscloudfront.GetDistributionByName(ctx, substrateCfg, naming.Substrate); err == nil {
		environment["DNS_DOMAIN_NAME"] = distribution.DomainName
	}

	// Construct the Intranet in every region we're using. For legacy reasons
	// this requires knowing the quality associated with the VPC created for
	// the Substrate account (from when it was called the admin account and
	// we had aspirations of supporting multiple of them).
	for _, region := range regions.Selected() {
		ui.Spinf("configuring the Substrate-managed Intranet in %s", region)
		cfg := substrateCfg.Regional(region)

		functionARN, err := awslambda.EnsureFunction(
			ctx,
			cfg,
			naming.Substrate,
			roleARN,
			environment,
			intranetzip.SubstrateIntranetZip,
		)
		ui.Must(err)
		//ui.Debug(functionARN)

		api, err := awsapigatewayv2.EnsureAPI(
			ctx,
			cfg,
			naming.Substrate,
			fmt.Sprintf("apigatewayv2.%s", dnsDomainName), // internal but safe DNS name (only 403 to 302 translation happens in CloudFront)
			aws.ToString(zone.Id),
			roleARN,
			functionARN,
		)
		ui.Must(err)
		//ui.Debug(api)

		_ /* authorizerId */, err = awsapigatewayv2.EnsureAuthorizer(ctx, cfg, api.Id, naming.Substrate, roleARN, functionARN)
		ui.Must(err)
		//ui.Debug(authorizerId)

		if err = awslambda.AddPermission(
			ctx,
			cfg,
			naming.Substrate,
			"apigateway.amazonaws.com",
			fmt.Sprintf(
				"arn:aws:execute-api:%s:%s:%s/*",
				region,
				cfg.MustAccountId(ctx),
				api.Id,
			),
		); awsutil.ErrorCodeIs(err, awslambda.ResourceConflictException) {
			err = nil // this is only safe because we've never changed the arguments to AddPermission
		}
		ui.Must(err)

		networks.ShareVPC(
			ctx,
			cfg,
			awscfg.Must(cfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour)).Regional(region),
			naming.Admin, naming.Admin, quality, // domain, environment, quality
			region,
		)

		ui.Stop("ok")
	}

	// Now configure CloudFront to handle redirects and front the API Gateways
	// in all our regions.
	//
	// It'd be great if, instead of having to drag the exp cookie around all
	// over everywhere, if we just had enough CPU or a reasonable enough input
	// data structure that we could do this:
	//
	//   if (Date.now() > JSON.parse(Buffer.from(event.request.cookies.id.value.split(".")[1], "base64url")).exp*1000) {
	//
	// That would be less of a hack.
	ui.Spin("configuring CloudFront for the Substrate-managed Intranet")
	distribution, err := awscloudfront.EnsureDistribution(
		ctx,
		substrateCfg,
		naming.Substrate,
		[]string{dnsDomainName},
		aws.ToString(zone.Id),
		[]awscloudfront.EventType{awscloudfront.ViewerRequest, awscloudfront.ViewerResponse},
		`
var querystring = require("querystring");

function handler(event) {
	if (event.context.eventType === "viewer-request") {

		try {
			if (Date.now() > parseInt(event.request.cookies.exp.value)*1000) {
				event.request.cookies = {hd: event.request.cookies.hd};
			}
		} catch (e) {
			//console.log(e);
			event.request.cookies = {hd: event.request.cookies.hd};
		}

		if (!event.request.cookies.a && event.request.uri !== "/credential-factory/fetch" && event.request.uri !== "/login") {
			var properties = Object.getOwnPropertyNames(event.request.querystring),
				query = {};
			for (var i = 0; i < properties.length; ++i) {
				query[properties[i]] = event.request.querystring[properties[i]].value;
			}
			return {
				headers: {location: {value: "/login?next=" + encodeURIComponent(
					event.request.uri + (
						properties.length === 0 ? "" : "?" + querystring.stringify(query)
					)
				)}},
				statusCode: 302,
				statusDescription: "Found"
			};
		}

		return event.request;
	} else if (event.context.eventType === "viewer-response") {

		event.response.headers["strict-transport-security"] = {value: "max-age=31536000; includeSubDomains; preload"};

		return event.response;
	}
}
		`,
		fmt.Sprintf("https://apigatewayv2.%s", dnsDomainName),
	)
	ui.Must(err)
	ui.Must(awsroute53.DeleteResourceRecordSets(ctx, substrateCfg, aws.ToString(zone.Id), func(record awsroute53.ResourceRecordSet) bool {
		recordName, preview := aws.ToString(record.Name), fmt.Sprintf("preview.%s.", dnsDomainName)
		return recordName == preview || strings.HasSuffix(recordName, "."+preview)
	}))
	ui.Stopf("distribution %s", distribution.Id)
	//ui.Debug(distribution)

	// Now that we have the CloudFront distribution for sure, reconfigure the
	// Lambda functions to make sure they know their DNS domain name.
	ui.Spin("connecting API Gateway v2 to CloudFront")
	environment["DNS_DOMAIN_NAME"] = dnsDomainName
	for _, region := range regions.Selected() {
		ui.Must2(awslambda.UpdateFunctionConfiguration(
			ctx,
			substrateCfg.Regional(region),
			naming.Substrate,
			roleARN,
			environment,
		))
	}
	ui.Stop("ok")

	// That's all for the Intranet itself.
	//
	// Get them setup to use Terraform in the Substrate account.

	// Copy module dependencies that are embedded in this binary into the
	// user's source tree.
	// TODO don't put these canned modules down if they're not already there
	intranetGlobalModule := terraform.IntranetGlobalModule()
	ui.Must(intranetGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/global")))
	intranetRegionalModule := terraform.IntranetRegionalModule()
	ui.Must(intranetRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "intranet/regional")))
	ui.Must(os.RemoveAll(filepath.Join(terraform.ModulesDirname, "intranet/regional/proxy")))
	lambdaFunctionGlobalModule := terraform.LambdaFunctionGlobalModule()
	ui.Must(lambdaFunctionGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "lambda-function/global")))
	lambdaFunctionRegionalModule := terraform.LambdaFunctionRegionalModule()
	ui.Must(lambdaFunctionRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "lambda-function/regional")))
	substrateGlobalModule := terraform.SubstrateGlobalModule()
	ui.Must(substrateGlobalModule.Write(filepath.Join(terraform.ModulesDirname, "substrate/global")))
	substrateRegionalModule := terraform.SubstrateRegionalModule()
	ui.Must(substrateRegionalModule.Write(filepath.Join(terraform.ModulesDirname, "substrate/regional")))

	tags := terraform.Tags{
		Domain:      Domain,
		Environment: Environment,
		Quality:     quality,
	}
	{
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, quality, regions.Global) // TODO "substrate" instead of Domain and quality if we're starting from scratch; still modules/intranet; prime it with useful data sources
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

		if *runTerraform {
			ui.Must(terraform.Init(dirname))
			if *providersLock {
				ui.Must(terraform.ProvidersLock(dirname))
			}
			if *noApply {
				ui.Must(terraform.Plan(dirname))
			} else {
				ui.Must(terraform.Apply(dirname, *autoApprove))
			}
		}
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, Domain, quality, region) // TODO "substrate" instead of Domain and quality if we're starting from scratch; still modules/intranet; prime it with useful data sources

		ui.Must(fileutil.Remove(filepath.Join(dirname, "network.tf")))

		file := terraform.NewFile()
		arguments := map[string]terraform.Value{
			"dns_domain_name":                    terraform.Q(dnsDomainName),
			"oauth_oidc_client_id":               terraform.Q(clientId),
			"oauth_oidc_client_secret_timestamp": terraform.Q(clientSecretTimestamp),
			"prefix":                             terraform.Q(naming.Prefix()),
			"selected_regions":                   terraform.QSlice(regions.Selected()),
			"stage_name":                         terraform.Q(quality),
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
		providersFile.Add(terraform.NetworkProviderFor(
			region,
			roles.ARN(networkCfg.MustAccountId(ctx), roles.NetworkAdministrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))

		if *runTerraform {
			ui.Must(terraform.Init(dirname))
			if *providersLock {
				ui.Must(terraform.ProvidersLock(dirname))
			}
			if *noApply {
				ui.Must(terraform.Plan(dirname))
			} else {
				ui.Must(terraform.Apply(dirname, *autoApprove))
			}
		}
	}

	return
}
