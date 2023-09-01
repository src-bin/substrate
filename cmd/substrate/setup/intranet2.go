package setup

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/src-bin/substrate/awsapigatewayv2"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awslambda"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awssecretsmanager"
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

	var telemetryYesNo string
	if telemetry.Enabled() {
		telemetryYesNo = "yes"
	} else {
		telemetryYesNo = "no"
	}
	for _, region := range regions.Selected() {
		ui.Debug(region)
		cfg := substrateCfg.Regional(region)

		functionARN, err := awslambda.EnsureFunction(
			ctx,
			cfg,
			naming.Substrate,
			roleARN,
			map[string]string{
				"AZURE_AD_TENANT_ID":                 tenantId,
				"OAUTH_OIDC_CLIENT_ID":               clientId,
				"OAUTH_OIDC_CLIENT_SECRET_TIMESTAMP": clientSecretTimestamp,
				"OKTA_HOSTNAME":                      hostname,
				"SELECTED_REGIONS":                   strings.Join(regions.Selected(), ","),
				"SUBSTRATE_PREFIX":                   naming.Prefix(),
				"SUBSTRATE_TELEMETRY":                telemetryYesNo,
			},
			intranetzip.SubstrateIntranetZip,
		)
		ui.Must(err)

		apiId, err := awsapigatewayv2.EnsureAPI(ctx, cfg, naming.Substrate, roleARN, functionARN)
		ui.Must(err)
		ui.Debug(apiId)

	}
	return "", "" // XXX causes tests to fail
}
