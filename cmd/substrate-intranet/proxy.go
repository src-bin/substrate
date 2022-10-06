package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/lambdautil"
)

// checkRedirect that doesn't redirect and instead lets clients do all the
// redirecting on their own, leaving this a request-at-a-time proxy.
func checkRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func init() {
	http.DefaultClient.CheckRedirect = checkRedirect
	http.DefaultClient.Timeout = 50 * time.Second // shorter than the Lambda function's so errors are visible
}

func proxy(
	ctx context.Context,
	event *events.APIGatewayProxyRequest,
) (*events.APIGatewayProxyResponse, error) {

	u, err := url.Parse(os.Getenv("PROXY_DESTINATION_URL"))
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	if proxyPathPrefix := os.Getenv("PROXY_PATH_PREFIX"); os.Getenv("STRIP_PATH_PREFIX") == "true" {
		u.Path = path.Join(u.Path, strings.TrimPrefix(event.Path, proxyPathPrefix))
	} else {
		u.Path = path.Join(u.Path, event.Path)
	}
	u.RawQuery = url.Values(event.MultiValueQueryStringParameters).Encode()
	//log.Printf("u: %s", u)

	body, err := lambdautil.EventBodyBuffer(event)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	//log.Printf("event.Body: %+v", body.String())

	req, err := http.NewRequest(event.HTTPMethod, u.String(), body)
	req.Header = event.MultiValueHeaders
	req.Header.Add("X-Forwarded-Host", event.Headers["Host"])
	req.Header.Del("X-Forwarded-Port") // we're on port 443
	req.Header.Add("X-Substrate-Intranet-Proxy-Principal", event.RequestContext.Authorizer[authorizerutil.PrincipalId].(string))
	log.Printf("req: %+v", req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	log.Printf("resp: %+v", resp)

	resp.Header.Del("Content-Length") // because we base64-encode
	// TODO reverse the Location header if there is one.
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body) // TODO get rid of this superfluous allocation by using body.Reset(), io.Copy, and base64.NewEncoder
	//log.Printf("len(b): %d, b: %#v", len(b), string(b))
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	s := base64.StdEncoding.EncodeToString(b)
	//log.Printf("len(s): %d, s: %#v", len(s), s)

	return &events.APIGatewayProxyResponse{
		Body:              s,
		IsBase64Encoded:   true,
		MultiValueHeaders: resp.Header,
		StatusCode:        resp.StatusCode,
	}, nil

}

// proxy2 is enough to prove that API Gateway v2 is now thankfully up to the
// task and doesn't strip trailing slashes.
func proxy2(ctx context.Context, event *events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	log.Printf("%+v\n", event)
	return &events.APIGatewayV2HTTPResponse{
		Body:       fmt.Sprintf("%+v\n", event),
		StatusCode: http.StatusOK,
	}, nil
}
