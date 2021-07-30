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
)

//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html

func indexHandler(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	var debug string
	if _, ok := event.QueryStringParameters["debug"]; ok {
		b, err := json.MarshalIndent(event, "", "\t")
		if err != nil {
			return nil, err
		}
		debug = string(b) + "\n" + strings.Join(os.Environ(), "\n")
	}

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}
	svc := apigateway.New(sess)

	out, err := svc.GetResources(&apigateway.GetResourcesInput{
		Limit:     aws.Int64(500),
		RestApiId: aws.String(event.RequestContext.APIID),
	})
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, item := range out.Items {
		if path := aws.StringValue(item.Path); strings.Count(path, "/") < 3 {
			switch path {

			// Paths that are not worth listing on the index.
			case "/":
			case "/credential-factory/authorize":
			case "/credential-factory/fetch":
			case "/login":

			default:
				paths = append(paths, path)
			}
		}
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
	handlers["/"] = indexHandler
}
