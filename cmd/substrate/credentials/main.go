package credentials

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main() {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatExport)
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	cmdutil.MustChdir()
	flag.Parse()
	version.Flag()
	if *quiet {
		ui.Quiet()
	}
	/*
		if awssts.CredentialFormatValid(*format) {
			ui.Fatalf(`-format="%s" not supported`, *format)
		}
	*/

	// Generate the token we'll exchange for AWS credentials.
	b := make([]byte, 48) // 48 binary bytes makes 64 base64-encoded bytes
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		ui.Fatal(err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)

	pathname, err := fileutil.PathnameInParents(choices.IntranetDNSDomainNameFilename)
	if err != nil {
		ui.Fatal("substrate.* not found in this or any parent directory; change to your Substrate repository or set SUBSTRATE_ROOT to its path in your environment")
	}
	intranetDNSDomainName, err := fileutil.ReadFile(pathname)
	if err != nil {
		ui.Fatal(err)
	}
	u := &url.URL{
		Scheme:   "https",
		Host:     fileutil.Tidy(intranetDNSDomainName),
		Path:     "/credential-factory/authorize",
		RawQuery: url.Values{"token": []string{token}}.Encode(),
	}
	ui.OpenURL(u.String())
	ui.Print("authenticate in your web browser if prompted, then return here")

	// Spin requesting /credentials/fetch?token=... until it responds 200 OK.
	ui.Spin("fetching credentials")
	u.Path = "/credential-factory/fetch"
	var credentials *sts.Credentials
	for range time.Tick(time.Second) {
		var err error
		credentials, err = fetch(u)
		if err != nil {
			ui.Fatal(err)
		}
		if credentials != nil {
			break
		}
	}
	ui.Stop("ok")

	// Print credentials in whatever format was requested.
	awssts.PrintCredentials(format, credentials)

}

func fetch(u *url.URL) (*sts.Credentials, error) {
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var credentials sts.Credentials
	if err := json.NewDecoder(resp.Body).Decode(&credentials); err != nil {
		return nil, err
	}
	return &credentials, nil
}
