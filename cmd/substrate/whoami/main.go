package whoami

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

var formatFlag = cmdutil.FormatFlag(
	cmdutil.FormatText,
	[]string{cmdutil.FormatEnv, cmdutil.FormatExport, cmdutil.FormatJSON, cmdutil.FormatText},
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami [--format <format>] [--quiet]",
		Short: "TODO whoami.Command().Short",
		Long:  `TODO whoami.Command().Long`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(awscfg.Main(cmd.Context()))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{"--format", "--quiet"}, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().Var(formatFlag, "format", formatFlag.Usage())
	cmd.RegisterFlagCompletionFunc("format", formatFlag.CompletionFunc)
	cmd.Flags().AddFlag(cmdutil.QuietFlag())
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	versionutil.WarnDowngrade(ctx, cfg)

	// TODO maintain a cache of account number, role name (or just role ARN), and tags by access key ID in .substrate.whoami.json; use that to make this fast enough to use in PS1

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	identity, err := cfg.Identity(ctx)
	ui.Must(err)

	switch formatFlag.String() {
	case cmdutil.FormatEnv:
		if identity.Tags.SubstrateSpecialAccount != "" {
			fmt.Printf(
				"ROLE=%q SUBSTRATE_SPECIAL_ACCOUNT=%q\n",
				identity.ARN,
				identity.Tags.SubstrateSpecialAccount,
			)
		} else {
			fmt.Printf(
				"DOMAIN=%q\nENVIRONMENT=%q\nQUALITY=%q\nROLE=%q\n",
				identity.Tags.Domain,
				identity.Tags.Environment,
				identity.Tags.Quality,
				identity.ARN,
			)
		}
	case cmdutil.FormatExport, cmdutil.FormatExportWithHistory:
		if identity.Tags.SubstrateSpecialAccount != "" {
			fmt.Printf(
				"export ROLE=%q SUBSTRATE_SPECIAL_ACCOUNT=%q\n",
				identity.ARN,
				identity.Tags.SubstrateSpecialAccount,
			)
		} else {
			fmt.Printf(
				"export DOMAIN=%q ENVIRONMENT=%q QUALITY=%q ROLE=%q\n",
				identity.Tags.Domain,
				identity.Tags.Environment,
				identity.Tags.Quality,
				identity.ARN,
			)
		}
	case cmdutil.FormatJSON:
		if identity.Tags.SubstrateSpecialAccount != "" {
			jsonutil.PrettyPrint(os.Stdout, map[string]string{
				"Role":                          identity.ARN,
				tagging.SubstrateSpecialAccount: identity.Tags.SubstrateSpecialAccount,
			})
		} else {
			jsonutil.PrettyPrint(os.Stdout, map[string]string{
				tagging.Domain:      identity.Tags.Domain,
				tagging.Environment: identity.Tags.Environment,
				tagging.Quality:     identity.Tags.Quality,
				"Role":              identity.ARN,
			})
		}
	case cmdutil.FormatText:
		if identity.Tags.SubstrateType == naming.Substrate {
			ui.Printf(
				"you're %s in your Substrate account",
				identity.ARN,
			)
		} else if identity.Tags.SubstrateSpecialAccount != "" {
			ui.Printf(
				"you're %s in your %s account",
				identity.ARN,
				identity.Tags.SubstrateSpecialAccount,
			)
		} else if identity.Tags.Domain == naming.Admin {
			ui.Printf(
				"you're %s in your admin account\nQuality: %s",
				identity.ARN,
				identity.Tags.Quality,
			)
		} else {
			ui.Printf(
				"you're %s in\nDomain:      %s\nEnvironment: %s\nQuality:     %s",
				identity.ARN,
				identity.Tags.Domain,
				identity.Tags.Environment,
				identity.Tags.Quality,
			)
		}
	default:
		ui.Fatalf("--format %q not supported", formatFlag)
	}

}
