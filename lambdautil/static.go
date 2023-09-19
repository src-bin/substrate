package lambdautil

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/oauthoidc"
)

func Static(contentType, body string) (*events.APIGatewayV2HTTPResponse, error) {
	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": contentType},
		StatusCode: http.StatusOK,
	}, nil
}

func StaticHandler(contentType, body string) func(
	context.Context,
	*awscfg.Config,
	*oauthoidc.Client,
	*events.APIGatewayProxyRequest,
) (*events.APIGatewayProxyResponse, error) {
	return func(
		context.Context,
		*awscfg.Config,
		*oauthoidc.Client,
		*events.APIGatewayProxyRequest,
	) (*events.APIGatewayProxyResponse, error) {
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": contentType},
			StatusCode: http.StatusOK,
		}, nil
	}
}
