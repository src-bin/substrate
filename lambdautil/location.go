package lambdautil

import (
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
)

func Location(event *events.APIGatewayV2HTTPRequest, query url.Values) string {
	u := &url.URL{
		Scheme:   "https",
		Path:     event.RawPath,
		RawQuery: query.Encode(),
	}
	if dnsDomainName := os.Getenv("DNS_DOMAIN_NAME"); dnsDomainName != "" {
		u.Host = dnsDomainName
	} else {
		u.Host = event.Headers["host"] // will this default confuse debugging?
	}
	return u.String()
}
