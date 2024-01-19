package substrate

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	switch event.RawPath {
	case "/substrate":
		return index(ctx, cfg, oc, event)
	case "/substrate/upgrade":
		return upgrade(ctx, cfg, oc, event)
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       fmt.Sprintf("%s not found\n", event.RawPath),
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusNotFound,
	}, nil
}

func index(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	upgradeVersion, _, err := versionutil.CheckForUpgrade()
	if err != nil {
		return nil, err
	}

	v := upgradeVersion
	if v == "" {
		v = version.Version
	}
	downloadURLs := []*url.URL{
		versionutil.DownloadURL(v, "darwin", "amd64"),
		versionutil.DownloadURL(v, "darwin", "arm64"),
		versionutil.DownloadURL(v, "linux", "amd64"),
		versionutil.DownloadURL(v, "linux", "arm64"),
	}

	body, err := lambdautil.RenderHTML(html, struct {
		Version, UpgradeVersion string
		DownloadURLs            []*url.URL
	}{
		Version:        version.Version,
		UpgradeVersion: upgradeVersion,
		DownloadURLs:   downloadURLs,
	})
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		StatusCode: http.StatusOK,
	}, nil
}

func upgrade(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {

	upgradeVersion, _, err := versionutil.CheckForUpgrade()
	if err != nil {
		return nil, err
	}
	if upgradeVersion == "" {
		return &events.APIGatewayV2HTTPResponse{
			Body: fmt.Sprintf("You're running Substrate %s, which is the latest version.", version.Version),
			Headers: map[string]string{
				"Content-Type": "text/plain",
				"Location":     "/substrate",
			},
			StatusCode: http.StatusFound,
		}, nil
	}

	if event.RequestContext.HTTP.Method == "POST" {
		return lambdautil.ErrorResponse2(errors.New("not implemented")) // TODO do the upgrade in a background Lambda
	}

	accountId, err := cfg.AccountId(ctx)
	if err != nil {
		return lambdautil.ErrorResponse2(err)
	}
	creds, err := awsiam.AllDayCredentials(
		ctx,
		cfg,
		accountId,
		fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.RoleName]),
	)
	if err != nil {
		return lambdautil.ErrorResponse2(err)
	}

	body, err := lambdautil.RenderHTML(htmlForUpgrade, struct {
		Version, UpgradeVersion string
		Credentials             aws.Credentials
	}{
		Version:        version.Version,
		UpgradeVersion: upgradeVersion,
		Credentials:    creds,
	})
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		StatusCode: http.StatusOK,
	}, nil
}

//go:embed substrate.html
var html string

//go:embed upgrade.html
var htmlForUpgrade string
