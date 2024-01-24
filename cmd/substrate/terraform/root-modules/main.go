package rootmodules

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/versionutil"
)

var format, formatFlag, formatCompletionFunc = cmdutil.FormatFlag(
	cmdutil.FormatText,
	[]cmdutil.Format{cmdutil.FormatJSON, cmdutil.FormatText},
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "root-modules [--format <format>] [--quiet]",
		Short: "enumerate root Terraform module directories",
		Long:  ``,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{"--format", "--quiet"}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().AddFlag(formatFlag)
	cmd.RegisterFlagCompletionFunc(formatFlag.Name, formatCompletionFunc)
	cmd.Flags().AddFlag(cmdutil.QuietFlag())
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {

	versionutil.WarnDowngrade(ctx, cfg)

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	cfg = awscfg.Must(cfg.OrganizationReader(ctx))
	ui.Must(cfg.ClearCachedAccounts())
	adminAccounts, serviceAccounts, substrateAccount, _, deployAccount, _, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		ui.Fatal(err)
	}

	var rootModules []string

	// The management and audit accounts don't run Terraform so they aren't
	// mentioned in this program's output.

	// Deploy account.
	if deployAccount != nil {
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			accounts.Deploy,
			regions.Global,
		))
		for _, region := range regions.Selected() {
			rootModules = append(rootModules, filepath.Join(
				terraform.RootModulesDirname,
				accounts.Deploy,
				region,
			))
		}
	}

	// Network account.
	if networkAccount != nil {
		veqpDoc, err := veqp.ReadDocument()
		if err != nil {
			ui.Fatal(err)
		}
		for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
			for _, region := range regions.Selected() {
				rootModules = append(rootModules, filepath.Join(
					terraform.RootModulesDirname,
					accounts.Network,
					eq.Environment,
					eq.Quality,
					region,
				))
			}
		}
	}

	// Admin accounts and the Substrate account that's taking their place.
	for _, account := range adminAccounts {
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			accounts.Admin,
			account.Tags[tagging.Quality],
			regions.Global,
		))
		for _, region := range regions.Selected() {
			rootModules = append(rootModules, filepath.Join(
				terraform.RootModulesDirname,
				accounts.Admin,
				account.Tags[tagging.Quality],
				region,
			))
		}
	}
	rootModules = append(rootModules, filepath.Join(
		terraform.RootModulesDirname,
		accounts.Admin,
		substrateAccount.Tags[tagging.Quality],
		regions.Global,
	))
	for _, region := range regions.Selected() {
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			accounts.Admin,
			substrateAccount.Tags[tagging.Quality],
			region,
		))
	}

	for _, account := range serviceAccounts {
		if _, ok := account.Tags[tagging.Domain]; !ok {
			continue
		}
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			account.Tags[tagging.Domain],
			account.Tags[tagging.Environment],
			account.Tags[tagging.Quality],
			regions.Global,
		))
		for _, region := range regions.Selected() {
			rootModules = append(rootModules, filepath.Join(
				terraform.RootModulesDirname,
				account.Tags[tagging.Domain],
				account.Tags[tagging.Environment],
				account.Tags[tagging.Quality],
				region,
			))
		}
	}

	switch *format {
	case cmdutil.FormatJSON:
		jsonutil.PrettyPrint(os.Stdout, rootModules)
	case cmdutil.FormatText:
		for _, rootModule := range rootModules {
			fmt.Println(rootModule)
		}
	default:
		ui.Fatal(cmdutil.FormatFlagError(*format))
	}

}
