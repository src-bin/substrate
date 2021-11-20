package main

import (
	"context"
	"encoding/base64"
	"errors"
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
	"github.com/src-bin/substrate/lambdautil"
)

// checkRedirect uses net/http's standard hook for managing HTTP redirects to
// propagate cookies even when differences between the domain of the original
// request's URL and the Location header suggest it should not be. This is safe
// in the context of this HTTP client because the destination is always
// declared up-front.
func checkRedirect(req *http.Request, via []*http.Request) error {

	// Copied from net/http's defaultCheckRedirect so we don't lose infinite
	// redirect suppression while adding cookie propagation.
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}

	// If the redirect is anything other than appending a '/' to the URI path,
	// let the user-agent handle it. We only even handle these redirects
	// because AWS API Gateway insists on stripping trailing slashes.
	//
	// TODO might need to reverse STRIP_PATH_PREFIX to be totally correct but
	// I don't see how to do it in Go's HTTP client.
	if req.URL.Path != via[0].URL.Path+"/" {
		return http.ErrUseLastResponse
	}
	log.Printf("redirecting internally to %s", req.URL)

	req.Method = via[0].Method
	req.Header = via[0].Header

	scheme := req.Header.Get("X-Forwarded-Proto")
	host := req.Header.Get("X-Forwarded-Host")
	port := req.Header.Get("X-Forwarded-Port")
	if port != "" && (scheme == "http" && port != "80" || scheme == "https" && port != "443") {
		host += ":" + port
	}
	if req.URL.Scheme == scheme && req.URL.Host == host {
		s := req.URL.String()
		req.URL.Scheme = via[0].URL.Scheme
		req.URL.Host = via[0].URL.Host
		log.Printf("rewriting internal redirect from %s to %s", s, req.URL)
	}

	return nil
}

func init() {
	http.DefaultClient.CheckRedirect = checkRedirect
	http.DefaultClient.Timeout = 50 * time.Second // shorter than the Lambda function's so errors are visible
}

func proxy(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

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
	log.Printf("u: %+v, u.String(): %s", u, u)

	body, err := lambdautil.EventBodyBuffer(event)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	//log.Printf("event.Body: %+v", body.String())

	req, err := http.NewRequest(event.HTTPMethod, u.String(), body)
	req.Header = event.MultiValueHeaders
	req.Header.Add("X-Forwarded-Host", event.Headers["Host"])
	req.Header.Del("X-Forwarded-Port") // we're on port 443
	req.Header.Add("X-Substrate-Intranet-Proxy-Principal", event.RequestContext.Authorizer["principalId"].(string))
	log.Printf("req: %+v", req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	log.Printf("resp: %+v", resp)

	resp.Header.Del("Content-Length") // because we base64-encode
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
