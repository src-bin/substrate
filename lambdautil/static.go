package lambdautil

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awscfg"
)

func StaticHandler(contentType, body string) func(context.Context, *awscfg.Config, *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	return func(context.Context, *awscfg.Config, *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": contentType},
			StatusCode: http.StatusOK,
		}, nil
	}
}
