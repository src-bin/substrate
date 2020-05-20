package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

		accessToken, err := oauthoidc.ParseAndVerifyJWT(doc.AccessToken, c)
		if err != nil {
			return errorResponse(err, doc.AccessToken), nil
		}
		idToken, err := oauthoidc.ParseAndVerifyJWT(doc.IDToken, c)
		if err != nil {
			return errorResponse(err, doc.IDToken), nil
		}
		return &events.APIGatewayProxyResponse{
			Body: fmt.Sprintf("%v\n\n%+v\n\n%s\n", accessToken, idToken, string(b)),
			Headers: map[string]string{
				"Content-Type": "text/plain",
				// "Location":     location,
				// "Set-Cookie":   cookie,
				// "Set-Cookie":   cookie,
			},
			StatusCode: http.StatusFound,
		}, nil
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
	q.Add("scope", "openid")
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

	c := oauthoidc.NewClient(
		"dev-662445.okta.com", // XXX
		oauthoidc.OktaPathQualifier("/oauth2/default/v1"),
		"0oacg1iawaojz8rOo4x6",                     // XXX
		"mFdL4HOHV5OquQVMm9SZd9r8RT9dLTccfTxPrfWc", // XXX
	)

	s := "eyJraWQiOiJ1N3BhQ2VBQ3RIaWUxa3BqSndMTWV2N0dXTEk4NmNyYXBrVXhxMDJKREswIiwiYWxnIjoiUlMyNTYifQ.eyJ2ZXIiOjEsImp0aSI6IkFULmVNcnBmaWpqM0duQUZxSzhjbjRFSEl1cWNla2l6TTJWd0RHcjRpNkFzVUEiLCJpc3MiOiJodHRwczovL2Rldi02NjI0NDUub2t0YS5jb20vb2F1dGgyL2RlZmF1bHQiLCJhdWQiOiJhcGk6Ly9kZWZhdWx0IiwiaWF0IjoxNTg5ODA2MDUwLCJleHAiOjE1ODk4MDk2NTAsImNpZCI6IjBvYWNnMWlhd2Fvano4ck9vNHg2IiwidWlkIjoiMDB1YXEwYXpzaWNyQzV2V1E0eDYiLCJzY3AiOlsib3BlbmlkIl0sInN1YiI6InNyYy1iaW4rdGVzdDFAcmNyb3dsZXkub3JnIn0.Vzzi2HrzofXkaSfPovitpOOrEFrHbKm381QRf33oYV_tXws1lNP4Ar6Wv6-06AdQ6OOdMHfnds0eyHK_6lT0F4hNl06RkStMpbaeMHrNbXZfWCoApcllg1607Mo8kyJGR7ZmciigiyxcfR9LsgtSdiJQ1Jna1LhHC9tKwGCgfkshbTY_fCsGX6HLP-MBsvCdSsFHcvlNhiySZHbboi7jQ_5Oq7Nz5ZyD6KJHIYL1uyoRRO99352kuocFQNRnx9OhXaHS5i0gehB-VthlLzX3LPoG7fZQkxMMkTrYuJSeKzO5SUCvDgwcTwHSwOV1Q6emD2cHIOiz1SfUI24herYq1Q"
	jwt, err := oauthoidc.ParseAndVerifyJWT(s, c)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Header: %+v\n", jwt.Header)
	fmt.Printf("Payload: %+v\n", jwt.Payload)
	fmt.Printf("Signature: %+v\n", jwt.Signature)
	fmt.Println("WE DID IT!")

	return

	lambda.Start(handle)
}
