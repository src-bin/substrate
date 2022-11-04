package credentials

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatExport)
	noOpen := flag.Bool("no-open", false, "do not try to open your web browser (so that you can copy the URL and open it yourself)")
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

	// Print credentials in whatever format was requested.
	awsutil.PrintCredentials(format, *creds)

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
