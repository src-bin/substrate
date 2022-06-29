package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/randutil"
	"github.com/src-bin/substrate/ui"
)

//go:generate go run ../../tools/template/main.go -name loginTemplate -package main login.html
//go:generate go run ../../tools/template/main.go -name redirectTemplate -package main redirect.html

const maxAge = 43200 // 12 hours, in seconds for the Max-Age modifier in the Set-Cookie header

func errorResponse(err error, extras ...interface{}) *events.APIGatewayProxyResponse {
	ui.PrintWithCaller(err) // log the error to CloudWatch but not the extras, which may be sensitive
	ss := make([]string, len(extras)+1)
	ss[0] = fmt.Sprintf("%v\n", err)
	for i, extra := range extras {
		var format string
		if _, ok := extra.([]byte); ok {
			format = "\n%s\n"
		} else {
			format = "\n%+v\n"
		}
		ss[i+1] = fmt.Sprintf(format, extra)
	}
	return &events.APIGatewayProxyResponse{
		Body:       strings.Join(ss, ""),
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusOK,
	}
}

func init() {
	handlers["/login"] = loginHandler
}

func loginHandler(ctx context.Context, cfg *awscfg.Config, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	// TODO logout per <https://developer.okta.com/docs/reference/api/oidc/#logout>

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	c, err := oauthoidc.NewClient(sess, event.StageVariables)
	if err != nil {
		return nil, err
	}

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
		doc := &oauthoidc.TokenResponse{}
		resp, tokenBody, err := c.Post(oauthoidc.Token, v, doc)
		if err != nil {
			return errorResponse(err, resp, tokenBody, doc), nil
		}
		idToken := &oauthoidc.IDToken{}
		if _, err := oauthoidc.ParseAndVerifyJWT(doc.IDToken, c, idToken); err != nil {
			return errorResponse(err, resp, tokenBody, doc), nil
		}
		if idToken.Nonce != state.Nonce {
			return errorResponse(oauthoidc.VerificationError{
				Field:    "nonce",
				Actual:   idToken.Nonce,
				Expected: state.Nonce,
			}, resp, tokenBody, doc), nil
		}

		var bodyV struct {
			IDToken  *oauthoidc.IDToken
			Location string
		}
		bodyV.IDToken = idToken
		multiValueHeaders := map[string][]string{
			"Content-Type": []string{"text/html"},
			"Set-Cookie": []string{
				fmt.Sprintf("a=%s; HttpOnly; Max-Age=%d; Secure", doc.AccessToken, maxAge),
				fmt.Sprintf("id=%s; HttpOnly; Max-Age=%d; Secure", doc.IDToken, maxAge),
				fmt.Sprintf("csrf=%s; HttpOnly; Max-Age=%d; Secure", randutil.String(), maxAge),
			},
		}
		bodyV.Location = state.Next
		if bodyV.Location == "" {
			bodyV.Location = "/"
		}
		multiValueHeaders["Location"] = []string{bodyV.Location}
		body, err := lambdautil.RenderHTML(loginTemplate(), bodyV)
		if err != nil {
			return nil, err
		}

		return &events.APIGatewayProxyResponse{
			Body:              body,
			MultiValueHeaders: multiValueHeaders,
			StatusCode:        http.StatusFound,
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
	scope := "openid email profile"
	if c.IsGoogle() {
		scope += " https://www.googleapis.com/auth/admin.directory.user.readonly"
	}
	q.Add("scope", scope)
	state = &oauthoidc.State{
		Next:  event.QueryStringParameters["next"],
		Nonce: nonce,
	}
	q.Add("state", state.String())

	var bodyV struct{ ErrorDescription, Location string }
	bodyV.ErrorDescription = event.QueryStringParameters["error_description"]
	headers := map[string]string{"Content-Type": "text/html"}
	statusCode := http.StatusOK
	bodyV.Location = c.URL(oauthoidc.Authorize, q).String()
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
