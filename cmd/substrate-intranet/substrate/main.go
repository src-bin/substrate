package substrate

import (
	"context"
	_ "embed"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awscfg"
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

//go:embed substrate.html
var html string
