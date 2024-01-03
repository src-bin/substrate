package login

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/randutil"
	"github.com/src-bin/substrate/ui"
)

//go:generate go run ../../../tools/template/main.go -name loginTemplate -package login login.html
//go:generate go run ../../../tools/template/main.go -name redirectTemplate -package login redirect.html

const maxAge = 43200 // 12 hours, in seconds for the Max-Age modifier in the Set-Cookie header

func errorResponse(err error, extras ...interface{}) *events.APIGatewayV2HTTPResponse {
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
	return &events.APIGatewayV2HTTPResponse{
		Body:       strings.Join(ss, ""),
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusOK,
	}
}

func Main(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	redirectURI := &url.URL{
		Path: event.RawPath,
		// no RawQuery because it's a redirect _URI_ not redirect _URL_
		Scheme: "https",
	}
	if dnsDomainName := os.Getenv("DNS_DOMAIN_NAME"); dnsDomainName != "" {
		redirectURI.Host = dnsDomainName
	} else {
		redirectURI.Host = event.Headers["host"] // will this default confuse debugging?
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
		resp, tokenBody, err := oc.Post(oauthoidc.Token, v, doc)
		if err != nil {
			return errorResponse(err, "oc.Post", resp, tokenBody, doc), nil
		}
		idToken := &oauthoidc.IDToken{}
		if _, err := oauthoidc.ParseAndVerifyJWT(doc.IDToken, oc, idToken); err != nil {
			return errorResponse(err, "oauthoidc.ParseAndVerifyJWT", resp, tokenBody, doc), nil
		}
		if idToken.Nonce != state.Nonce {
			return errorResponse(oauthoidc.VerificationError{
				Field:    "nonce",
				Actual:   idToken.Nonce,
				Expected: state.Nonce,
			}, "idToken.Nonce != state.Nonce", resp, tokenBody, doc), nil
		}

		var bodyV struct {
			IDToken  *oauthoidc.IDToken
			Location string
		}
		bodyV.IDToken = idToken
		setCookies := []string{
			fmt.Sprintf("a=%s; HttpOnly; Max-Age=%d; Secure", doc.AccessToken, maxAge),
			fmt.Sprintf("exp=%d; HttpOnly; Max-Age=%d; Secure", idToken.Expires, maxAge),
			fmt.Sprintf("id=%s; HttpOnly; Max-Age=%d; Secure", doc.IDToken, maxAge),
			fmt.Sprintf("csrf=%s; HttpOnly; Max-Age=%d; Secure", randutil.String(), maxAge),
		}
		if i := strings.LastIndexByte(idToken.Email, '@'); i != -1 && oc.IsGoogle() {
			setCookies = append(setCookies, fmt.Sprintf(
				"hd=%s; HttpOnly; Max-Age=%d; Secure",
				idToken.Email[i+1:], // just the domain from the email address
				10*365*86400,        // 10 years-ish
			))
		}
		bodyV.Location = state.Next
		if bodyV.Location == "" {
			bodyV.Location = "/"
		}
		body, err := lambdautil.RenderHTML(loginTemplate(), bodyV)
		if err != nil {
			return nil, err
		}

		return &events.APIGatewayV2HTTPResponse{
			Body:    body,
			Cookies: setCookies,
			Headers: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
				"Location":     bodyV.Location,
			},
			StatusCode: http.StatusFound,
		}, nil
	}

	q := url.Values{}
	q.Add("client_id", oc.ClientId)
	nonce, err := oauthoidc.Nonce()
	if err != nil {
		return nil, err
	}
	q.Add("nonce", nonce)
	q.Add("redirect_uri", redirectURI.String())
	q.Add("response_type", "code")
	scope := "openid email profile"
	if oc.IsAzureAD() {
		scope += " CustomSecAttributeAssignment.Read.All User.Read"
	}
	if oc.IsGoogle() {
		scope += " https://www.googleapis.com/auth/admin.directory.user.readonly"
		if hd := lambdautil.Cookie2(event.Cookies, "hd"); hd != nil {
			q.Add("hd", hd.Value)
		}
	}
	if oc.IsOkta() {
		scope += " okta.users.read.self"
	}
	q.Add("scope", scope)
	state = &oauthoidc.State{
		Next:  event.QueryStringParameters["next"],
		Nonce: nonce,
	}
	q.Add("state", state.String())

	var bodyV struct{ ErrorDescription, Location string }
	bodyV.ErrorDescription = event.QueryStringParameters["error_description"]
	headers := map[string]string{"Content-Type": "text/html; charset=utf-8"}
	statusCode := http.StatusOK
	bodyV.Location = oc.URL(oauthoidc.Authorize, q).String()
	if bodyV.ErrorDescription == "" {
		headers["Location"] = bodyV.Location
		statusCode = http.StatusFound
	}
	body, err := lambdautil.RenderHTML(redirectTemplate(), bodyV)
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    headers,
		StatusCode: statusCode,
	}, nil
}
