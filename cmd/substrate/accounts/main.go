package accounts

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatText) // default to undocumented special value
	number := flag.String("number", "", `with -format "json", account number of the single AWS account to output`)
	onlyTags := flag.Bool("only-tags", false, `with -format "json" and -number "...", output only the tags on the account`)
	cmdutil.MustChdir()
	flag.Usage = func() {
		ui.Print("Usage: substrate accounts [-format <format>] [-number <number>] [-only-tags]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	cfg = awscfg.Must(cfg.OrganizationReader(ctx))

	adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		ui.Fatal(err)
	}
	switch format.String() {

	case cmdutil.SerializationFormatJSON:

		// Maybe only print one account.
		prettyPrintJSON := func(account *awsorgs.Account) {
			if *onlyTags {
				ui.PrettyPrintJSON(account.Tags)
			} else {
				ui.PrettyPrintJSON(account)
			}
		}
		if *number == aws.StringValue(managementAccount.Id) {
			prettyPrintJSON(managementAccount)
			return
		} else if *number == aws.StringValue(auditAccount.Id) {
			prettyPrintJSON(auditAccount)
			return
		} else if *number == aws.StringValue(networkAccount.Id) {
			prettyPrintJSON(networkAccount)
			return
		} else if *number == aws.StringValue(deployAccount.Id) {
			prettyPrintJSON(deployAccount)
			return
		}
		for _, account := range adminAccounts {
			if *number == aws.StringValue(account.Id) {
				prettyPrintJSON(account)
				return
			}
		}
		for _, account := range serviceAccounts {
			if *number == aws.StringValue(account.Id) {
				prettyPrintJSON(account)
				return
			}
		}

		// We're still here so print them all.
		ui.PrettyPrintJSON(append(append([]*awsorgs.Account{
			managementAccount,
			auditAccount,
			networkAccount,
			deployAccount,
		}, adminAccounts...), serviceAccounts...))

	case cmdutil.SerializationFormatShell:
		fmt.Println("substrate bootstrap-management-account")
		fmt.Println("substrate bootstrap-network-account")
		fmt.Println("substrate bootstrap-deploy-account")
		for _, account := range adminAccounts {
			fmt.Printf(
				"substrate create-admin-account -quality %q\n",
				account.Tags[tags.Quality],
			)
		}
		for _, account := range serviceAccounts {
			if _, ok := account.Tags[tags.Domain]; !ok {
				continue
			}
			fmt.Printf(
				"substrate create-account -domain %q -environment %q -quality %q\n",
				account.Tags[tags.Domain],
				account.Tags[tags.Environment],
				account.Tags[tags.Quality],
			)
		}

	case cmdutil.SerializationFormatText:
		f, err := os.Open(accounts.CheatSheetFilename)
		if err != nil {
			ui.Fatal(err)
		}
		defer f.Close()
		io.Copy(os.Stdout, f)

	default:
		ui.Fatalf("-format %q not supported", format)
	}

	// Update substrate.accounts.txt unconditionally as this is the expected
	// side-effect of running this program.
	if err := accounts.CheatSheet(ctx, cfg); err != nil {
		ui.Fatal(err)
	}

}
