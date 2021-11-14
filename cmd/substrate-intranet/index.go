package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
)

//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html

// unlistedPaths are specific complete paths that are not worth listing in the
// index. There are also patterns that are made unlisted below.
var unlistedPaths = []string{
	"/",
	"/credential-factory/authorize",
	"/credential-factory/fetch",
	"/login",
}

func indexHandler(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}
	svc := apigateway.New(sess)

	var debug string
	if _, ok := event.QueryStringParameters["debug"]; ok {
		b, err := json.MarshalIndent(event, "", "\t")
		if err != nil {
			return nil, err
		}
		debug += string(b) + "\n" + strings.Join(os.Environ(), "\n") + "\n"

		c, err := oauthoidc.NewClient(sess, event.StageVariables)
		if err != nil {
			return nil, err
		}
		if c.IsGoogle() {
			c.AccessToken = event.RequestContext.Authorizer["AccessToken"].(string)
			body, err := oauthoidc.GoogleAdminDirectoryUser(
				c,
				event.RequestContext.Authorizer["principalId"].(string),
			)
			if err != nil {
				return nil, err
			}
			b, err := json.MarshalIndent(body, "", "\t")
			if err != nil {
				return nil, err
			}
			debug += string(b) + "\n"
		}
	}

	out, err := svc.GetResources(&apigateway.GetResourcesInput{
		Limit:     aws.Int64(500),
		RestApiId: aws.String(event.RequestContext.APIID),
	})
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, item := range out.Items {
		path := aws.StringValue(item.Path)
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
	handlers["/"] = indexHandler
}
