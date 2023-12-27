package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/randutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

var (
	format, formatFlag, formatCompletionFunc = cmdutil.FormatFlag(
		cmdutil.FormatExport,
		[]cmdutil.Format{cmdutil.FormatEnv, cmdutil.FormatExport, cmdutil.FormatJSON},
	)
	force, noOpen = new(bool), new(bool)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials [--format <format>] [--force] [--no-open] [--quiet]",
		Short: "TODO credentials.Command().Short",
		Long:  `TODO credentials.Command().Long`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--format",
				"--force",
				"--no-open",
				"--quiet",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().AddFlag(formatFlag)
	cmd.RegisterFlagCompletionFunc(formatFlag.Name, formatCompletionFunc)
	cmd.Flags().BoolVar(force, "force", false, "force minting new credentials even if there are valid credentials in the environment")
	cmd.Flags().BoolVar(noOpen, "no-open", false, "do not try to open your web browser (so that you can copy the URL and open it yourself)")
	cmd.Flags().AddFlag(cmdutil.QuietFlag())
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {

	if !*force {
		if _, err := cfg.GetCallerIdentity(ctx); err == nil {
			expiry, err := time.Parse(time.RFC3339, os.Getenv(cmdutil.SUBSTRATE_CREDENTIALS_EXPIRATION))
			if err != nil {
				ui.Print("found valid credentials in the environment; exiting without minting new ones")
				cfg.Telemetry().PostWait(ctx)
				return
			}
			if time.Now().Add(6 * time.Hour).Before(expiry) {
				hours := expiry.Sub(time.Now()).Hours()
				ui.Printf(
					"found credentials in the environment that are still valid for more than %d hours; exiting without minting new ones",
					int(hours),
				)
				cfg.Telemetry().PostWait(ctx)
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
	defer cfg.Telemetry().Wait(ctx)

	versionutil.WarnDowngrade(ctx, cfg)

	// Print credentials in whatever format was requested.
	cmdutil.PrintCredentials(*format, *creds)

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
