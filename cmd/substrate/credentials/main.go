package credentials

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/features"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/randutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	force := flag.Bool("force", false, "force minting new credentials even if there are valid credentials in the environment")
	format := cmdutil.SerializationFormatFlag(
		cmdutil.SerializationFormatExport,
		cmdutil.SerializationFormatUsage,
	)
	noOpen := flag.Bool("no-open", false, "do not try to open your web browser (so that you can copy the URL and open it yourself)")
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	flag.Usage = func() {
		ui.Print("Usage: substrate credentials [-format <format>] [-quiet]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *quiet {
		ui.Quiet()
	}

	if !*force {
		if _, err := cfg.GetCallerIdentity(ctx); err == nil {
			expiry, err := time.Parse(time.RFC3339, os.Getenv(cmdutil.SUBSTRATE_CREDENTIALS_EXPIRATION))
			if err != nil {
				ui.Print("found valid credentials in the environment; exiting without minting new ones")
				return
			}
			if time.Now().Add(6 * time.Hour).Before(expiry) {
				hours := expiry.Sub(time.Now()).Hours()
				ui.Printf(
					"found credentials in the environment that are still valid for more than %d hours; exiting without minting new ones",
					int(hours),
				)
				return
			}
		}
	}

	// Generate the token we'll exchange for AWS credentials.
	token := randutil.String()

	u := &url.URL{
		Scheme:   "https",
		Host:     naming.MustIntranetDNSDomainName(),
		Path:     "/credential-factory/authorize",
		RawQuery: url.Values{"token": []string{token}}.Encode(),
	}
	if *noOpen {
		ui.Printf("open <%s> in your web browser; authenticate if prompted, then return here", u)
	} else {
		ui.OpenURL(u.String())
		ui.Print("authenticate in your web browser, if prompted, then return here")
	}

	// Spin requesting /credentials/fetch?token=... until it responds 200 OK.
	ui.Spin("fetching credentials")
	u.Path = "/credential-factory/fetch"
	ch := time.After(time.Hour)
	var creds *aws.Credentials
	for range time.Tick(time.Second) {
		select {
		case <-ch:
			ui.Stop("timed out")
			return
		default:
		}
		var err error
		creds, err = fetch(u)
		if err != nil {
			ui.Fatal(err)
		}
		if creds != nil {
			break
		}
	}
	ui.Stop("ok")

	cfg.SetCredentials(ctx, *creds)
	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	versionutil.WarnDowngrade(ctx, cfg)

	if features.MacOSKeychain.Enabled() {
		ui.Must(cmdutil.SetTPM(*creds))
	}

	// Print credentials in whatever format was requested.
	cmdutil.PrintCredentials(format, *creds)

}

func fetch(u *url.URL) (*aws.Credentials, error) {
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	//log.Printf("<%s> resp: %+v", u.String(), resp)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	//log.Printf("<%s> body: %s", u.String(), string(body))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, nil
	} else if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s from <%s>: %s", resp.Status, u, string(body))
	}
	var creds aws.Credentials
	if err := json.Unmarshal(body, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}
