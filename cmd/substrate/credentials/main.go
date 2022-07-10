package credentials

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/randutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatExport)
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	cmdutil.MustChdir()
	flag.Usage = func() {
		ui.Print("Usage: substrate credentials [-format <format>] [-quiet]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *quiet {
		ui.Quiet()
	}

	// Generate the token we'll exchange for AWS credentials.
	token := randutil.String()

	pathname, err := fileutil.PathnameInParents(naming.IntranetDNSDomainNameFilename)
	if err != nil {
		ui.Fatalf("substrate.* not found in this or any parent directory; change to your Substrate repository or set SUBSTRATE_ROOT to its path in your environment (%v)", err)
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
	ui.Print("authenticate in your web browser, if prompted, then return here")

	// Spin requesting /credentials/fetch?token=... until it responds 200 OK.
	ui.Spin("fetching credentials")
	u.Path = "/credential-factory/fetch"
	var credentials *aws.Credentials
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

	cfg.SetCredentials(ctx, *credentials)
	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	// Print credentials in whatever format was requested.
	awsutil.PrintCredentials(format, *credentials)

}

func fetch(u *url.URL) (*aws.Credentials, error) {
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var credentials aws.Credentials
	if err := json.NewDecoder(resp.Body).Decode(&credentials); err != nil {
		return nil, err
	}
	return &credentials, nil
}
