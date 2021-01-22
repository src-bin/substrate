package main

import (
	"context"
	"net/http"
	"sort"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/lambdautil"
)

//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

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
	var root string
	for _, item := range out.Items {
		if item.ParentId == nil {
			root = aws.StringValue(item.Id)
			break
		}
	}
	paths := make([]string, 0)
	for _, item := range out.Items {
		if aws.StringValue(item.ParentId) == root {
			paths = append(paths, aws.StringValue(item.Path))
		}
	}
	sort.Strings(paths)

	body, err := lambdautil.RenderHTML(indexTemplate(), struct{ Paths []string }{paths})
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	lambda.Start(handle)
}
