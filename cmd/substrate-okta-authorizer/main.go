package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/oauthoidc"
)

func handle(ctx context.Context, event *events.APIGatewayCustomAuthorizerRequestTypeRequest) (*events.APIGatewayCustomAuthorizerResponse, error) {

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	clientSecret, err := awssecretsmanager.CachedSecret(
		secretsmanager.New(sess),
		fmt.Sprintf(
			"OktaClientSecret-%s",
			event.StageVariables["OktaClientID"],
		),
		event.StageVariables["OktaClientSecretTimestamp"],
	)
	if err != nil {
		return nil, err
	}

	c := oauthoidc.NewClient(
		event.StageVariables["OktaHostname"],
		oauthoidc.OktaPathQualifier("/oauth2/default"),
		event.StageVariables["OktaClientID"],
		clientSecret,
	)

	accessToken := &oauthoidc.OktaAccessToken{}
	idToken := &oauthoidc.OktaIDToken{}
	req := &http.Request{Header: http.Header{
		"Cookie": event.MultiValueHeaders["cookie"], // beware the case-sensitivity
	}}
	for _, cookie := range req.Cookies() {
		switch cookie.Name {
		case "a":
			if _, err := oauthoidc.ParseAndVerifyJWT(cookie.Value, c, accessToken); err != nil {
				return nil, err
			}
		case "id":
			if _, err := oauthoidc.ParseAndVerifyJWT(cookie.Value, c, idToken); err != nil {
				return nil, err
			}
		}
	}

	context := map[string]interface{}{}
	if context["AccessToken"], err = accessToken.JSONString(); err != nil {
		return nil, err
	}
	if context["IDToken"], err = idToken.JSONString(); err != nil {
		return nil, err
	}
	return &events.APIGatewayCustomAuthorizerResponse{
		Context: context,
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
