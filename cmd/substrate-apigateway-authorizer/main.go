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
	"github.com/src-bin/substrate/policies"
)

func handle(ctx context.Context, event *events.APIGatewayCustomAuthorizerRequestTypeRequest) (*events.APIGatewayCustomAuthorizerResponse, error) {

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	clientSecret, err := awssecretsmanager.CachedSecret(
		secretsmanager.New(sess),
		fmt.Sprintf(
			"%s-%s",
			oauthoidc.OAuthOIDCClientSecret,
			event.StageVariables[oauthoidc.OAuthOIDCClientID],
		),
		event.StageVariables[oauthoidc.OAuthOIDCClientSecretTimestamp],
	)
	if err != nil {
		return nil, err
	}

	var pathQualifier oauthoidc.PathQualifier
	if hostname := event.StageVariables[oauthoidc.OktaHostname]; hostname == oauthoidc.OktaHostnameValueForGoogleIDP {
		pathQualifier = oauthoidc.GooglePathQualifier()
	} else {
		pathQualifier = oauthoidc.OktaPathQualifier(hostname, "default")
	}
	c := oauthoidc.NewClient(
		pathQualifier,
		event.StageVariables[oauthoidc.OAuthOIDCClientID],
		clientSecret,
	)

	context := map[string]interface{}{
		"Location": "/login?next=" + event.Path, // where API Gateway will send the browser when unauthorized
	}

	idToken := &oauthoidc.IDToken{}
	req := &http.Request{Header: http.Header{
		"Cookie": event.MultiValueHeaders["cookie"], // beware the case-sensitivity
	}}
	for _, cookie := range req.Cookies() {
		switch cookie.Name {
		case "a":
			context["AccessToken"] = cookie.Value
		case "id":
			if _, err := oauthoidc.ParseAndVerifyJWT(cookie.Value, c, idToken); err != nil {
				context["Error"] = err
				idToken = &oauthoidc.IDToken{} // revert to zero-value and thus to denying access
				continue
			}
			if context["IDToken"], err = idToken.JSONString(); err != nil {
				return nil, err
			}
		}
	}

	effect := policies.Deny
	if idToken.Email != "" {
		effect = policies.Allow
	}
	return &events.APIGatewayCustomAuthorizerResponse{
		Context: context,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Statement: []events.IAMPolicyStatement{{
				Action:   []string{"execute-api:Invoke"},
				Effect:   string(effect),
				Resource: []string{event.MethodArn},
			}},
			Version: "2012-10-17",
		},
		PrincipalID: idToken.Email,
	}, nil
}

func main() {
	lambda.Start(handle)
}
