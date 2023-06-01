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
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
)

//go:generate go run ../../../tools/template/main.go -name indexTemplate index.html

// unlistedPaths are specific complete paths that are not worth listing in the
// index. There are also patterns that are made unlisted below.
var unlistedPaths = []string{
	"/",
	"/audit", // TODO make it skip paths that don't respond to GET requests instead of having to enumerate this
	"/credential-factory/authorize",
	"/credential-factory/fetch",
	"/favicon.ico",
	"/js",
	"/js/accounts.js",
	"/login",
}

func Main(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayProxyRequest,
) (*events.APIGatewayProxyResponse, error) {

	var debug string
	if _, ok := event.QueryStringParameters["debug"]; ok {
		b, err := json.MarshalIndent(event, "", "\t")
		if err != nil {
			return nil, err
		}
		debug += string(b) + "\n" + strings.Join(os.Environ(), "\n") + "\n"
	}

	out, err := cfg.APIGateway().GetResources(ctx, &apigateway.GetResourcesInput{
		Limit:     aws.Int32(500),
		RestApiId: aws.String(event.RequestContext.APIID),
	})
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, item := range out.Items {
		path := aws.ToString(item.Path)
		if strings.Contains(path, "{") {
			continue
		}
		if strings.Count(path, "/") >= 3 {
			continue
		}
		if i := sort.SearchStrings(unlistedPaths, path); i < len(unlistedPaths) && unlistedPaths[i] == path {
			continue
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

	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil
}

func init() {
	sort.Strings(unlistedPaths)
}
