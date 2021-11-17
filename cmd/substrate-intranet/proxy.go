package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

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

	req.Header.Add("Cookie", via[0].Header.Get("Cookie"))
	// FIXME also need to modify req.URL to switch the Intranet host for the PROXY_DESTINATION_URL host (and port) plus implement STRIP_PATH_PREFIX again
	///*
	log.Printf("req: %+v", req)
	for i, v := range via {
		log.Printf("via[%d]: %+v", i, v)
	}
	//*/
	return nil
}

func init() {
	http.DefaultClient.CheckRedirect = checkRedirect
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
	// TODO figure out why trailing slashes are stripped and whether it matters. It matters. It breaks Jenkins with path_part="jenkins" and --prefix="/jenkins" at /jenkins/ and /jenkins/view/all/ (so it's not just the root).

	body, err := lambdautil.EventBodyBuffer(event)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	log.Printf("event.Body: %+v", body.String())

	req, err := http.NewRequest(event.HTTPMethod, u.String(), body)
	req.Header = event.MultiValueHeaders
	req.Header.Add("X-Forwarded-Host", event.Headers["Host"])
	//req.Header.Add("X-Forwarded-Proto", "https") // already added by API Gateway
	req.Header.Add("X-Substrate-Intranet-Proxy-Principal", event.RequestContext.Authorizer["principalId"].(string))
	log.Printf("req: %+v", req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	log.Printf("resp: %+v", resp)
	resp.Header.Del("Content-Length") // because we base64-encode
	defer resp.Body.Close()
	body.Reset()
	if _, err := io.Copy(base64.NewEncoder(base64.StdEncoding, body), resp.Body); err != nil {
		return lambdautil.ErrorResponse(err)
	}
	log.Printf("body: %+v", body.String())

	return &events.APIGatewayProxyResponse{
		Body:              body.String(),
		IsBase64Encoded:   true,
		MultiValueHeaders: resp.Header,
		StatusCode:        resp.StatusCode,
	}, nil

}
