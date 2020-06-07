package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/src-bin/substrate/lambdautil"
)

//go:generate go run ../../tools/template/main.go -name instanceFactoryTemplate -package main instance_factory.html

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	// DEBUG
	b, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return nil, err
	}

	body, err := lambdautil.RenderHTML(instanceFactoryTemplate(), struct {
		Debug string
	}{
		Debug: string(b),
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

func main() {
	lambda.Start(handle)
}
