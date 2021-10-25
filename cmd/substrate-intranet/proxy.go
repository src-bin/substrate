package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// TODO make a proxy that can be placed at any old path in an API Gateway
// TODO put it in the admin VPC (with the same quality as this Intranet)
// TODO assume it'll be able to reach anything it needs to since everything's peered with admin VPCs
// TODO configure destinations using aws_api_gateway_integration resources and their request_parameters attributes to inject headers to control routing

func proxy(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	b, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return nil, err
	}
	body := string(b) + "\n" + strings.Join(os.Environ(), "\n")

	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusOK,
	}, nil
	/*
		return &events.APIGatewayProxyResponse{
			Body:       "404 Not Found\n",
			Headers:    map[string]string{"Content-Type": "text/plain"},
			StatusCode: http.StatusNotFound,
		}, nil
	*/
}
