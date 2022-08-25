package federation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
)

// ConsoleSigninURL exchanges a set of STS credentials for a signin token that
// grants the opener access to the AWS Console per the algorithm outlined in
// <https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_enable-console-custom-url.html>.
// The event parameter may be nil, in which case we attempt to learn the
// issuing URL by reading substrate.intranet-dns-domain-name from disk.
func ConsoleSigninURL(
	credentials aws.Credentials,
	destination string,
	event *events.APIGatewayProxyRequest,
) (string, error) {

	// Step 1: AssumeRole, which is technically optional, as all that's really
	// required is a set of credentials.

	// Step 2: Exchange the credentials for a signin token.
	u := &url.URL{
		Scheme: "https",
		Host:   "signin.aws.amazon.com",
		Path:   "/federation",
		RawQuery: url.Values{
			"Action": []string{"getSigninToken"},
			"Session": []string{jsonutil.MustString(struct {
				ID    string `json:"sessionId"`
				Key   string `json:"sessionKey"`
				Token string `json:"sessionToken"`
			}{
				credentials.AccessKeyID,
				credentials.SecretAccessKey,
				credentials.SessionToken,
			})},
			// "SessionDuration": []string{"3599"}, // minimum 900, maximum is however long than you have left in the role you've assumed
		}.Encode(),
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	var body struct{ SigninToken string }
	if err := json.Unmarshal(b, &body); err != nil {
		//log.Print(string(b)) // it'll be a bunch of HTML with a generic error
		return "", err
	}

	// Step 3: Construct the console signin URL. Try slightly to come up with
	// good defaults for destination and issuer URLs.
	if destination == "" {
		destination = "https://console.aws.amazon.com/"
	}
	issuer := "https://src-bin.com/substrate/"
	if event != nil {
		issuer = fmt.Sprintf("https://%s/", event.RequestContext.DomainName)
	} else if intranetDNSDomainName, err := fileutil.ReadFile(naming.IntranetDNSDomainNameFilename); err == nil {
		issuer = fmt.Sprintf("https://%s/", intranetDNSDomainName)
	}
	u.RawQuery = url.Values{
		"Action":      []string{"login"},
		"Destination": []string{destination}, // must be on aws.amazon.com or console.aws.amazon.com (signin.aws.amazon.com, in particular, isn't acceptable so no logout via login)
		"Issuer":      []string{issuer},
		"SigninToken": []string{body.SigninToken},
	}.Encode()

	// Step 4: Bounce through a URL like <https://signin.aws.amazon.com/oauth?Action=logout&redirect_uri=https://aws.amazon.com>
	// to logout of any existing session before logging in, which AWS won't do
	// automatically and which is really annoying to make users do manually.
	/*
		redirectURI := u.String()
		u.Path = "/oauth"
		u.RawQuery = url.Values{
			"Action":       []string{"logout"},
			"redirect_uri": []string{redirectURI},
		}.Encode()
	*/

	return u.String(), nil
}
