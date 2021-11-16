package main

import (
	"bytes"
	"context"
	"encoding/base64"
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

func proxy(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	u, err := url.Parse(os.Getenv("PROXY_DESTINATION_URL"))
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	uOriginal := u.String()
	_ = uOriginal
	if proxyPathPrefix := os.Getenv("PROXY_PATH_PREFIX"); os.Getenv("STRIP_PATH_PREFIX") == "true" {
		u.Path = path.Join(u.Path, strings.TrimPrefix(event.Path, proxyPathPrefix))
	} else {
		u.Path = path.Join(u.Path, event.Path)
	}
	// TODO figure out why trailing slashes are stripped and whether it matters. It matters. It breaks Jenkins with path_part="jenkins" and --prefix="/jenkins" at /jenkins/ and /jenkins/view/all/ (so it's not just the root).

	var body *bytes.Buffer
	if event.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}
		body = bytes.NewBuffer(b)
	} else {
		body = bytes.NewBufferString(event.Body)
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
