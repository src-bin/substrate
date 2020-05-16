package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handle(ctx context.Context, event *events.APIGatewayCustomAuthorizerRequest) (*events.APIGatewayCustomAuthorizerResponse, error) {
	log.Printf("%+v", event)
	return &events.APIGatewayCustomAuthorizerResponse{
		Context: map[string]interface{}{},
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   "Allow",
					Resource: []string{event.MethodArn},
				},
			},
			Version: "2012-10-17",
		},
		PrincipalID:        "rcrowley",
		UsageIdentifierKey: "rcrowley-was-here",
	}, nil
}

func main() {
	lambda.Start(handle)
}
