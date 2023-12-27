package list

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	setupcloudtrail "github.com/src-bin/substrate/cmd/substrate/setup-cloudtrail"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

var (
	format, formatFlag, formatCompletionFunc = cmdutil.FormatFlag(
		cmdutil.FormatText,
		[]cmdutil.Format{cmdutil.FormatJSON, cmdutil.FormatShell, cmdutil.FormatText},
	)
	number                                    = new(string)
	onlyTags                                  = new(bool)
	refresh                                   = new(bool)
	autoApprove, noApply, ignoreServiceQuotas = new(bool), new(bool), new(bool)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: `list [--format <format>] [--number <number>] [--only-tags] [--refresh]
  substrate account list --format shell [--auto-approve|--no-apply] [--ignore-service-quotas] [--refresh]`,
		Short: "TODO list.Command().Short",
		Long:  `TODO list.Command().Long`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--format",
				"--number", "--only-tags",
				"--refresh",
				"--auto-approve", "--no-apply", "--ignore-service-quotas",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().AddFlag(formatFlag)
	cmd.RegisterFlagCompletionFunc(formatFlag.Name, formatCompletionFunc)
	cmd.Flags().StringVar(number, "number", "", "with --format json, account number of the single AWS account to output")
	cmd.RegisterFlagCompletionFunc("number", cmdutil.NoCompletionFunc)
	cmd.Flags().BoolVar(onlyTags, "only-tags", false, "with --format json and --number <number>, output only the tags on the account")
	cmd.Flags().BoolVar(refresh, "refresh", false, "clear Substrate's local cache of AWS accounts and refresh it from the AWS Organizations API")
	cmd.Flags().BoolVar(autoApprove, "auto-approve", false, "with --format shell, add the --auto-approve flag to all the generated commands that accept it")
	cmd.Flags().BoolVar(noApply, "no-apply", false, "with --format shell, add the --no-apply flag to all the generated commands that accept it")
	cmd.Flags().BoolVar(ignoreServiceQuotas, "ignore-service-quotas", false, "with --format shell, add the --ignore-service-quotas flag to all the generated commands that accept it")
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	versionutil.WarnDowngrade(ctx, cfg)

	if *refresh {
		ui.Must(cfg.ClearCachedAccounts())
	}

	// Update substrate.accounts.txt unconditionally as this is the expected
	// side-effect of running this program.
	ui.Must(accounts.CheatSheet(ctx, cfg))

	adminAccounts, serviceAccounts, substrateAccount, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	ui.Must(err)
	switch *format {

	case cmdutil.FormatJSON:

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

	case cmdutil.FormatShell:
		var autoApproveFlag, ignoreServiceQuotasFlag, noApplyFlag string
		if *autoApprove {
			autoApproveFlag = " --auto-approve" // leading space to format pleasingly both ways
		}
		if *ignoreServiceQuotas {
			ignoreServiceQuotasFlag = " --ignore-service-quotas" // leading space to format pleasingly both ways
		}
		if *noApply {
			noApplyFlag = " --no-apply" // leading space to format pleasingly both ways
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
				"substrate account update%s%s%s --domain %q --environment %q --quality %q\n",
				autoApproveFlag,
				ignoreServiceQuotasFlag,
				noApplyFlag,
				account.Tags[tagging.Domain],
				account.Tags[tagging.Environment],
				account.Tags[tagging.Quality],
			)
		}

	case cmdutil.FormatText:
		f, err := os.Open(accounts.CheatSheetFilename)
		if err != nil {
			ui.Fatal(err)
		}
		defer f.Close()
		io.Copy(os.Stdout, f)

	default:
		ui.Fatal(cmdutil.FormatFlagError(*format))
	}

}
