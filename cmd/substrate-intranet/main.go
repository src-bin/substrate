package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Handler func(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)

var handlers = map[string]Handler{}

func main() {
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
}
