package index

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awsapigatewayv2"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
)

//go:generate go run ../../../tools/template/main.go -name indexTemplate index.html

func Main(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {

	var debug string
	if _, ok := event.QueryStringParameters["debug"]; ok {
		b, err := json.MarshalIndent(event, "", "\t")
		if err != nil {
			return nil, err
		}
		debug += string(b) + "\n" + strings.Join(os.Environ(), "\n") + "\n"
	}

	routes, err := awsapigatewayv2.GetRoutes(ctx, cfg, event.RequestContext.APIID)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, route := range routes {
		path := aws.ToString(route.RouteKey)
		if strings.HasPrefix(path, "ANY ") || strings.HasPrefix(path, "GET ") { // don't list POST-only routes because what would we POST?
			path = path[4:]
		} else {
			continue // unlists "$default", which is most of the Substrate-managed Intranet
		}
		if path == "/credential-factory/fetch" || path == "/favicon.ico" || path == "/login" {
			continue // unlists the bits of the Substrate-managed Intranet that don't require auth[nz]
		}
		if strings.Contains(path, "{") {
			continue // unlists parameterized paths we can't meaningfully link to
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	body, err := lambdautil.RenderHTML(indexTemplate(), struct {
		Debug string
		Paths []string
	}{
		Debug: debug,
		Paths: paths,
	})
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		StatusCode: http.StatusOK,
	}, nil
}
