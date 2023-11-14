package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/ui"
)

//go:generate go run ../../tools/dispatch-map/main.go .
//go:generate go run ../../tools/dispatch-map/main.go -function JavaScript -o dispatch-map-js.go .
//go:generate go run ../../tools/dispatch-map/main.go -function Main2 -o dispatch-map-main.go .

func main() {
	ctx := contextutil.WithValues(context.Background(), "substrate-intranet", "", "")

	cfg, err := awscfg.NewConfig(ctx)
	if err != nil {
		ui.Fatal(err)
	}

	clientId := os.Getenv(oauthoidc.OAuthOIDCClientId)
	var pathQualifier oauthoidc.PathQualifier
	switch oauthoidc.IdPName(clientId) {
	case oauthoidc.AzureAD:
		pathQualifier = oauthoidc.AzureADPathQualifier(os.Getenv(oauthoidc.AzureADTenantId))
	case oauthoidc.Google:
		pathQualifier = oauthoidc.GooglePathQualifier()
	case oauthoidc.Okta:
		pathQualifier = oauthoidc.OktaPathQualifier(os.Getenv(oauthoidc.OktaHostname))
	}
	oc, err := oauthoidc.NewClient(
		ctx,
		cfg,
		clientId,
		os.Getenv(oauthoidc.OAuthOIDCClientSecretTimestamp),
		pathQualifier,
	)
	if err != nil {
		ui.Fatal(err)
	}

	lambda.Start(&Mux{
		Authorizer: authorizer2(cfg, oc),
		Handler: func(ctx context.Context, event *events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
			var principalId string
			if event.RequestContext.Authorizer != nil {
				principalId = fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.PrincipalId])
			}
			ctx = contextutil.WithValues(ctx, "substrate-intranet", event.RawPath, principalId)
			ui.Printf("%s %s %s", event.RequestContext.HTTP.Method, event.RawPath, principalId)

			if event.RawPath == "/favicon.ico" {
				return &events.APIGatewayV2HTTPResponse{StatusCode: http.StatusNoContent}, nil
			} else if path.Dir(event.RawPath) == "/js" && path.Ext(event.RawPath) == ".js" {
				k := strings.TrimSuffix(path.Base(event.RawPath), ".js")
				if f, ok := dispatchMapJavaScript[k]; ok {
					return f(ctx, cfg, oc.Copy(), event)
				}
			} else {
				k := strings.SplitN(event.RawPath, "/", 3)[1] // safe because there's always at least the leading '/'
				if k == "" {
					k = "index"
				}
				if f, ok := dispatchMapMain2[k]; ok {
					return f(ctx, cfg, oc.Copy(), event)
				}
			}

			return &events.APIGatewayV2HTTPResponse{
				Body:       fmt.Sprintf("%s not found\n", event.RawPath),
				Headers:    map[string]string{"Content-Type": "text/plain"},
				StatusCode: http.StatusNotFound,
			}, nil
		},
	})
}
