package setup

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awsapigatewayv2"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscloudfront"
	"github.com/src-bin/substrate/awslambda"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awsutil"
	intranetzip "github.com/src-bin/substrate/cmd/substrate/intranet-zip"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/ui"
)

const (
	AzureADTenantFilename = "substrate.azure-ad-tenant"
	OktaHostnameFilename  = "substrate.okta-hostname"
	SAMLMetadataFilename  = "substrate.saml-metadata.xml"

	OAuthOIDCClientIdFilename              = "substrate.oauth-oidc-client-id"
	OAuthOIDCClientSecretTimestampFilename = "substrate.oauth-oidc-client-secret-timestamp"
)

func intranet2(ctx context.Context, mgmtCfg, substrateCfg *awscfg.Config) (dnsDomainName string, idpName oauthoidc.Provider) {
	roleARN := roles.ARN(substrateCfg.MustAccountId(ctx), roles.Substrate)

	// Make arrangements for a hosted zone to appear in this account so that
	// the Intranet can configure itself.  It's possible to do this entirely
	// programmatically but there's a lot of UI surface area involved in doing
	// a really good job.
	// TODO allow them to just use the API Gateway or CloudFront hostname.
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

	// Prepare the Lambda function's environment variables. If we can find our
	// CloudFront distribution, include its DNS domain name in the environment
	// right away to prevent a transient outage while we reconfigure the Lambda
	// function twice, once before managing the CloudFront distribution and
	// once again after. On first runs, the Lambda function will be partially
	// configured before the CloudFront distribution is created and then
	// reconfigured to include the CloudFront distribution's DNS domain name.
	var telemetryYesNo string
	if telemetry.Enabled() {
		telemetryYesNo = "yes"
	} else {
		telemetryYesNo = "no"
	}
	environment := map[string]string{
		"AZURE_AD_TENANT_ID":                 tenantId,
		"OAUTH_OIDC_CLIENT_ID":               clientId,
		"OAUTH_OIDC_CLIENT_SECRET_TIMESTAMP": clientSecretTimestamp,
		"OKTA_HOSTNAME":                      hostname,
		"SELECTED_REGIONS":                   strings.Join(regions.Selected(), ","),
		"SUBSTRATE_PREFIX":                   naming.Prefix(),
		"SUBSTRATE_TELEMETRY":                telemetryYesNo,
	}
	if distribution, err := awscloudfront.GetDistributionByName(ctx, substrateCfg, naming.Substrate); err == nil {
		environment["DNS_DOMAIN_NAME"] = distribution.DomainName
	}

	// Construct the Intranet in every region we're using.
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
				event.request.cookies = {};
			}
		} catch (e) {
			//console.log(e);
			event.request.cookies = {};
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
	ui.Must(awsroute53.DeleteResourceRecordSets(ctx, cfg, aws.ToString(zone.Id), func(record awsroute53.ResourceRecordSet) bool {
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

	return dnsDomainName, idpName
}
