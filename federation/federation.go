package federation

import (
	"encoding/json"
	"fmt"
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
			// "SessionDuration": []string{"600"}, // it breaks if this is uncommented, with seemingly any value
		}.Encode(),
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct{ SigninToken string }
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
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
		"Destination": []string{destination},
		"Issuer":      []string{issuer},
		"SigninToken": []string{body.SigninToken},
	}.Encode()

	return u.String(), nil
}
