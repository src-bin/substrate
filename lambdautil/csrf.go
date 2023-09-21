package lambdautil

import (
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
)

const (
	CookieName = "csrf"
	FieldName  = "csrf"
)

func CSRFCookie(event *events.APIGatewayProxyRequest) string {
	req := &http.Request{Header: http.Header{
		"Cookie": event.MultiValueHeaders["cookie"], // beware the case-sensitivity
	}}
	for _, cookie := range req.Cookies() {
		if cookie.Name == CookieName {
			return cookie.Value
		}
	}
	return ""
}

func CSRFCookie2(event *events.APIGatewayV2HTTPRequest) string {
	cookie := Cookie2(event.Cookies, CookieName)
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

type CSRFError struct{}

func (CSRFError) Error() string {
	return "potential CSRF detected"
}

func PreventCSRF(body url.Values, event *events.APIGatewayProxyRequest) error {
	csrf := CSRFCookie(event)
	if csrf == "" {
		return CSRFError{}
	}
	if body.Get(FieldName) == csrf {
		return nil
	}
	return CSRFError{}
}

func PreventCSRF2(body url.Values, event *events.APIGatewayV2HTTPRequest) error {
	csrf := CSRFCookie2(event)
	if csrf == "" {
		return CSRFError{}
	}
	if body.Get(FieldName) == csrf {
		return nil
	}
	return CSRFError{}
}
