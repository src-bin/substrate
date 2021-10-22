package awssts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/fileutil"
)

// ConsoleSigninURL exchanges a set of STS credentials for a signin token that
// grants the opener access to the AWS Console per the algorithm outlined in
// <https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_enable-console-custom-url.html>.
func ConsoleSigninURL(
	svc *sts.STS,
	credentials *sts.Credentials,
	destination string,
) (string, error) {

	// Step 1: AssumeRole, which is technically optional, as all that's really
	// required is a set of credentials.

	// Step 2: Exchange the credentials for a signin token.
	session, err := json.Marshal(struct {
		ID    string `json:"sessionId"`
		Key   string `json:"sessionKey"`
		Token string `json:"sessionToken"`
	}{
		aws.StringValue(credentials.AccessKeyId),
		aws.StringValue(credentials.SecretAccessKey),
		aws.StringValue(credentials.SessionToken),
	})
	if err != nil {
		return "", err
	}
	u := &url.URL{
		Scheme: "https",
		Host:   "signin.aws.amazon.com",
		Path:   "/federation",
		RawQuery: url.Values{
			"Action":  []string{"getSigninToken"},
			"Session": []string{string(session)},
			// "SessionDuration": []string{"600"}, // FIXME it breaks if this is uncommented, with seemingly any value
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

	// Step 3: Construct the console signin URL.
	if destination == "" {
		destination = "https://console.aws.amazon.com/"
	}
	intranetDNSDomainName, err := fileutil.ReadFile(choices.IntranetDNSDomainNameFilename)
	var issuer string
	if err != nil {
		issuer = "https://src-bin.com/substrate/"
	} else {
		issuer = fmt.Sprintf("https://%s/login", intranetDNSDomainName)
	}
	u.RawQuery = url.Values{
		"Action":      []string{"login"},
		"Destination": []string{destination},
		"Issuer":      []string{issuer},
		"SigninToken": []string{body.SigninToken},
	}.Encode()

	return u.String(), nil
}
