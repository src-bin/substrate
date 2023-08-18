package setup

import (
	"context"

	"github.com/src-bin/substrate/awsapigatewayv2"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awslambda"
	intranetzip "github.com/src-bin/substrate/cmd/substrate/intranet-zip"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

func intranet2(ctx context.Context, mgmtCfg, substrateCfg *awscfg.Config) (dnsDomainName string, idpName oauthoidc.Provider) {
	roleARN := roles.ARN(substrateCfg.MustAccountId(ctx), roles.Substrate)
	for _, region := range regions.Selected() {
		ui.Debug(region)
		cfg := substrateCfg.Regional(region)

		environment := map[string]string{
			"AZURE_AD_TENANT_ID":                 "TODO",
			"OAUTH_OIDC_CLIENT_ID":               "TODO",
			"OAUTH_OIDC_CLIENT_SECRET_TIMESTAMP": "TODO",
			"OKTA_HOSTNAME":                      "TODO",
			"SELECTED_REGIONS":                   "TODO",
			"SUBSTRATE_PREFIX":                   "TODO",
			"SUBSTRATE_TELEMETRY":                "TODO",
		}
		functionARN, err := awslambda.EnsureFunction(
			ctx,
			cfg,
			naming.Substrate,
			roleARN,
			environment,
			intranetzip.SubstrateIntranetZip,
		)
		ui.Must(err)

		apiId, err := awsapigatewayv2.EnsureAPI(ctx, cfg, naming.Substrate, roleARN, functionARN)
		ui.Must(err)
		ui.Debug(apiId)

	}
	return "", "" // XXX causes tests to fail
}
