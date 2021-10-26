package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// TODO refactor this program to use the dispatchMap pattern from cmd/substrate.

type Handler func(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)

var handlers = map[string]Handler{}

func main() {
	const (
		IntranetFunctionName                     = "Intranet"
		IntranetAPIGatewayAuthorizerFunctionName = "IntranetAPIGatewayAuthorizer"
		IntranetProxyFunctionNamePrefix          = "IntranetProxy-"
		varName                                  = "AWS_LAMBDA_FUNCTION_NAME"
	)
	functionName := os.Getenv(varName)

	if functionName == IntranetAPIGatewayAuthorizerFunctionName /* begin remove in 2021.11 */ || functionName == "substrate-apigateway-authorizer" /* end remove in 2021.11 */ {
		lambda.Start(authorizer)

	} else if functionName == IntranetFunctionName /* begin remove in 2021.11 */ || functionName == "substrate-intranet" /* end remove in 2021.11 */ {
		lambda.Start(func(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
			if h, ok := handlers[event.Path]; ok {
				return h(ctx, event)
			}
			return &events.APIGatewayProxyResponse{
				Body:       "404 Not Found\n",
				Headers:    map[string]string{"Content-Type": "text/plain"},
				StatusCode: http.StatusNotFound,
			}, nil
		})

	} else if strings.HasPrefix(functionName, IntranetProxyFunctionNamePrefix) {
		//pathPart := strings.TrimPrefix(functionName, IntranetProxyFunctionNamePrefix)
		lambda.Start(proxy)

	} else {
		lambda.Start(func(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
			return &events.APIGatewayProxyResponse{
				Body:       fmt.Sprintf("500 Internal Server Error\n\n%s=\"%s\" is not an expected configuration\n", varName, functionName),
				Headers:    map[string]string{"Content-Type": "text/plain"},
				StatusCode: http.StatusInternalServerError,
			}, nil
		})

	}
}
