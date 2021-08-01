package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
)

type Handler func(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)

var handlers = map[string]Handler{}

func main() {
	const varName = "AWS_LAMBDA_FUNCTION_NAME"
	switch functionName := os.Getenv(varName); functionName {

	case "substrate-apigateway-authorizer":
		lambda.Start(func(ctx context.Context, event *events.APIGatewayCustomAuthorizerRequestTypeRequest) (*events.APIGatewayCustomAuthorizerResponse, error) {

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

			u := &url.URL{
				Path:     event.Path,
				RawQuery: url.Values(event.MultiValueQueryStringParameters).Encode(),
			}
			next := u.String()
			u.Path = "/login"
			u.RawQuery = url.Values{"next": []string{next}}.Encode()
			context := map[string]interface{}{
				"Location": u.String(), // where API Gateway will send the browser when unauthorized
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
		})

	case "substrate-intranet":
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

	default:
		lambda.Start(func(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
			return &events.APIGatewayProxyResponse{
				Body:       fmt.Sprintf("500 Internal Server Error\n\n%s=\"%s\" is not an expected configuration\n", varName, functionName),
				Headers:    map[string]string{"Content-Type": "text/plain"},
				StatusCode: http.StatusInternalServerError,
			}, nil
		})

	}
}
