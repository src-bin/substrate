package accounts

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	setupcloudtrail "github.com/src-bin/substrate/cmd/substrate/setup-cloudtrail"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	autoApprove := flag.Bool("auto-approve", false, `with -format "shell", add the -auto-approve flag to all the generated commands that accept it`)
	format := cmdutil.SerializationFormatFlag(
		cmdutil.SerializationFormatText,
		`output format - "text" for human-readable plaintext, "json" for output like the AWS organizations:ListAccounts API augmented with Substrate roles and tags, or "shell" for a shell program that will update all your AWS accounts`,
	)
	ignoreServiceQuotas := flag.Bool("ignore-service-quotas", false, `with -format "shell", add the -ignore-service-quotas flag to all the generated commands that accept it`)
	noApply := flag.Bool("no-apply", false, `with -format "shell", add the -no-apply flag to all the generated commands that accept it`)
	number := flag.String("number", "", `with -format "json", account number of the single AWS account to output`)
	onlyTags := flag.Bool("only-tags", false, `with -format "json" and -number "...", output only the tags on the account`)
	flag.Usage = func() {
		ui.Print("Usage: substrate accounts [-format <format>] [-number <number>] [-only-tags]")
		ui.Print("Usage: substrate accounts -format shell [-auto-approve|-no-apply]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	cfg = awscfg.Must(cfg.OrganizationReader(ctx))
	versionutil.WarnDowngrade(ctx, cfg)

	// Update substrate.accounts.txt unconditionally as this is the expected
	// side-effect of running this program.
	ui.Must(accounts.CheatSheet(ctx, cfg))

	if *number == "" {
		ui.Must(cfg.ClearCachedAccounts())
	}
	adminAccounts, serviceAccounts, substrateAccount, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	ui.Must(err)
	switch format.String() {

	case cmdutil.SerializationFormatJSON:

		// Maybe only print one account.
		prettyPrintJSON := func(account *awsorgs.Account) {
			if *onlyTags {
				jsonutil.PrettyPrint(os.Stdout, account.Tags)
			} else {
				jsonutil.PrettyPrint(os.Stdout, account)
			}
		}
		if *number == aws.ToString(managementAccount.Id) {
			prettyPrintJSON(managementAccount)
			return
		} else if *number == aws.ToString(auditAccount.Id) {
			prettyPrintJSON(auditAccount)
			return
		} else if *number == aws.ToString(networkAccount.Id) {
			prettyPrintJSON(networkAccount)
			return
		} else if *number == aws.ToString(deployAccount.Id) {
			prettyPrintJSON(deployAccount)
			return
		} else if *number == aws.ToString(substrateAccount.Id) {
			prettyPrintJSON(substrateAccount)
			return
		}
		for _, account := range adminAccounts {
			if *number == aws.ToString(account.Id) {
				prettyPrintJSON(account)
				return
			}
		}
		for _, account := range serviceAccounts {
			if *number == aws.ToString(account.Id) {
				prettyPrintJSON(account)
				return
			}
		}

		// We're still here so print them all.
		jsonutil.PrettyPrint(os.Stdout, append(append([]*awsorgs.Account{
			managementAccount,
			auditAccount,
			networkAccount,
			deployAccount,
			substrateAccount,
		}, adminAccounts...), serviceAccounts...))

	case cmdutil.SerializationFormatShell:
		var autoApproveFlag, ignoreServiceQuotasFlag, noApplyFlag string
		if *autoApprove {
			autoApproveFlag = " -auto-approve" // leading space to format pleasingly both ways
		}
		if *ignoreServiceQuotas {
			ignoreServiceQuotasFlag = " -ignore-service-quotas" // leading space to format pleasingly both ways
		}
		if *noApply {
			noApplyFlag = " -no-apply" // leading space to format pleasingly both ways
		}

		fmt.Println("set -e -x")

		fmt.Printf("substrate setup%s%s%s\n", autoApproveFlag, ignoreServiceQuotasFlag, noApplyFlag)

		if ok, err := ui.ConfirmFile(setupcloudtrail.ManageCloudTrailFilename); err != nil {
			ui.Fatal(err)
		} else if ok {
			fmt.Print("substrate setup-cloudtrail\n")
		}

		for _, account := range serviceAccounts {
			if _, ok := account.Tags[tagging.Domain]; !ok {
				continue
			}
			fmt.Printf(
				"substrate create-account%s%s%s -domain %q -environment %q -quality %q\n",
				autoApproveFlag,
				ignoreServiceQuotasFlag,
				noApplyFlag,
				account.Tags[tagging.Domain],
				account.Tags[tagging.Environment],
				account.Tags[tagging.Quality],
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

}
