package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/src-bin/substrate/oauthoidc"
)

func handle(ctx context.Context, event *events.APIGatewayCustomAuthorizerRequest) (*events.APIGatewayCustomAuthorizerResponse, error) {
	log.Printf("%+v", event)

	c := oauthoidc.NewClient(
		"dev-662445.okta.com", // XXX
		oauthoidc.OktaPathQualifier("/oauth2/default"),
		"0oacg1iawaojz8rOo4x6",                     // XXX
		"mFdL4HOHV5OquQVMm9SZd9r8RT9dLTccfTxPrfWc", // XXX
	)

	aCookie, idCookie := "", "" // TODO
	accessToken := &oauthoidc.OktaAccessToken{}
	if _, err := oauthoidc.ParseAndVerifyJWT(aCookie, c, accessToken); err != nil {
		return nil, err
	}
	idToken := &oauthoidc.OktaIDToken{}
	if _, err := oauthoidc.ParseAndVerifyJWT(idCookie, c, idToken); err != nil {
		return nil, err
	}

	return &events.APIGatewayCustomAuthorizerResponse{
		Context: map[string]interface{}{
			"foo": "bar", // XXX
		},
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
		PrincipalID: accessToken.Subject,
	}, nil
}

func main() {
	lambda.Start(handle)
}
