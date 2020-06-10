package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
)

//go:generate go run ../../tools/template/main.go -name loginTemplate -package main login.html
//go:generate go run ../../tools/template/main.go -name redirectTemplate -package main redirect.html

func errorResponse(err error, s string) *events.APIGatewayProxyResponse {
	log.Printf("%+v", err)
	return &events.APIGatewayProxyResponse{
		Body:       err.Error() + "\n\n" + s + "\n",
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusOK,
	}
}

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	// TODO logout per <https://developer.okta.com/docs/reference/api/oidc/#logout>

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	clientSecret, err := awssecretsmanager.CachedSecret(
		secretsmanager.New(sess),
		fmt.Sprintf(
			"OAuthOIDCClientSecret-%s",
			event.StageVariables["OAuthOIDCClientID"],
		),
		event.StageVariables["OAuthOIDCClientSecretTimestamp"],
	)
	if err != nil {
		return nil, err
	}

	c := oauthoidc.NewClient(
		event.StageVariables["OktaHostname"],
		oauthoidc.OktaPathQualifier("/oauth2/default"),
		event.StageVariables["OAuthOIDCClientID"],
		clientSecret,
	)
	redirectURI := &url.URL{
		Scheme: "https",
		Host:   event.Headers["Host"],
		Path:   event.Path,
	}

	code := event.QueryStringParameters["code"]
	state, err := oauthoidc.ParseState(event.QueryStringParameters["state"])
	if err != nil {
		return nil, err
	}
	if code != "" && state != nil {
		v := url.Values{}
		v.Add("code", code)
		v.Add("grant_type", "authorization_code")
		v.Add("redirect_uri", redirectURI.String())
		doc := &oauthoidc.OktaTokenResponse{}
		if _, err := c.Post(oauthoidc.TokenPath, v, doc); err != nil {
			return nil, err
		}

		accessToken := &oauthoidc.OktaAccessToken{}
		if _, err := oauthoidc.ParseAndVerifyJWT(doc.AccessToken, c, accessToken); err != nil {
			return errorResponse(err, doc.AccessToken), nil
		}
		idToken := &oauthoidc.OktaIDToken{}
		if _, err := oauthoidc.ParseAndVerifyJWT(doc.IDToken, c, idToken); err != nil {
			return errorResponse(err, doc.IDToken), nil
		}
		if idToken.Nonce != state.Nonce {
			return errorResponse(oauthoidc.VerificationError{"nonce", idToken.Nonce, state.Nonce}, doc.IDToken), nil
		}

		var bodyV struct {
			AccessToken *oauthoidc.OktaAccessToken
			IDToken     *oauthoidc.OktaIDToken
			Location    string
		}
		bodyV.AccessToken = accessToken
		bodyV.IDToken = idToken
		multiValueHeaders := map[string][]string{
			"Content-Type": []string{"text/html"},
			"Set-Cookie": []string{
				fmt.Sprintf("a=%s; HttpOnly; Max-Age=43200; Secure", doc.AccessToken),
				fmt.Sprintf("id=%s; HttpOnly; Max-Age=43200; Secure", doc.IDToken),
			},
		}
		statusCode := http.StatusOK
		if state.Next != "" {
			bodyV.Location = state.Next
			multiValueHeaders["Location"] = []string{state.Next}
			statusCode = http.StatusFound
		}
		body, err := lambdautil.RenderHTML(loginTemplate(), bodyV)
		if err != nil {
			return nil, err
		}

		return &events.APIGatewayProxyResponse{
			Body:              body,
			MultiValueHeaders: multiValueHeaders,
			StatusCode:        statusCode,
		}, nil
	}

	q := url.Values{}
	q.Add("client_id", c.ClientID)
	nonce, err := oauthoidc.Nonce()
	if err != nil {
		return nil, err
	}
	q.Add("nonce", nonce)
	q.Add("redirect_uri", redirectURI.String())
	q.Add("response_type", "code")
	q.Add("scope", "openid profile") // TODO figure out how to get "preferred_username", too
	state = &oauthoidc.State{
		Next:  event.QueryStringParameters["next"],
		Nonce: nonce,
	}
	q.Add("state", state.String())

	var bodyV struct{ ErrorDescription, Location string }
	bodyV.ErrorDescription = event.QueryStringParameters["error_description"]
	headers := map[string]string{"Content-Type": "text/html"}
	statusCode := http.StatusOK
	bodyV.Location = c.URL(oauthoidc.AuthorizePath, q).String()
	if bodyV.ErrorDescription == "" {
		headers["Location"] = bodyV.Location
		statusCode = http.StatusFound
	}
	body, err := lambdautil.RenderHTML(redirectTemplate(), bodyV)
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    headers,
		StatusCode: statusCode,
	}, nil
}

func main() {
	lambda.Start(handle)
}
