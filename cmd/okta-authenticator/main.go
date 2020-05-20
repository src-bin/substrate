package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/src-bin/substrate/oauthoidc"
)

func errorResponse(err error, s string) *events.APIGatewayProxyResponse {
	return &events.APIGatewayProxyResponse{
		Body:       err.Error() + "\n\n" + s + "\n",
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusOK,
	}
}

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	b, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return nil, err
	}

	// TODO logout per <https://developer.okta.com/docs/reference/api/oidc/#logout>

	c := oauthoidc.NewClient(
		"dev-662445.okta.com", // XXX
		oauthoidc.OktaPathQualifier("/oauth2/default/v1"),
		"0oacg1iawaojz8rOo4x6",                     // XXX
		"mFdL4HOHV5OquQVMm9SZd9r8RT9dLTccfTxPrfWc", // XXX
	)

	code := event.QueryStringParameters["code"]
	state := event.QueryStringParameters["state"]
	if code != "" && state != "" {
		v := url.Values{}
		v.Add("code", code)
		v.Add("grant_type", "authorization_code")
		v.Add("redirect_uri", "https://czo8u1t120.execute-api.us-west-1.amazonaws.com/alpha/login") // XXX
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
		return &events.APIGatewayProxyResponse{
			Body: fmt.Sprintf("%+v\n\n%+v\n\n%s\n", accessToken, idToken, string(b)),
			Headers: map[string]string{
				"Content-Type": "text/plain",
				// "Location":     location,
				// "Set-Cookie":   cookie,
				// "Set-Cookie":   cookie,
			},
			StatusCode: http.StatusFound,
		}, nil
	}

	error_description := event.QueryStringParameters["error_description"]
	if error_description != "" {
		// TODO
	}

	q := url.Values{}
	q.Add("client_id", c.ClientID)
	nonce, err := oauthoidc.Nonce()
	if err != nil {
		return nil, err
	}
	q.Add("nonce", nonce)
	q.Add("redirect_uri", "https://czo8u1t120.execute-api.us-west-1.amazonaws.com/alpha/login") // XXX
	q.Add("response_type", "code")
	q.Add("scope", "openid") // TODO figure out how to get the "profile" scope, too, because we need preferred_username
	q.Add("state", "foobar") // XXX
	location := c.URL(oauthoidc.AuthorizePath, q).String()

	return &events.APIGatewayProxyResponse{
		Body: `<!DOCTYPE html>
<html lang="en">
<meta charset="utf-8">
<title>Intranet</title>
<body>
<h1>Intranet</h1>
<p>Redirecting to <a href="` + location + `">Okta</a>.</p>
<hr>
<pre>` + string(b) + `</pre>
</body>
</html>
`,
		Headers: map[string]string{
			"Content-Type": "text/html",
			// "Location":     location,
		},
		StatusCode: http.StatusFound,
	}, nil
}

func main() {
	lambda.Start(handle)
}
