package accounts

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/ui"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:    "accounts (deprecated)",
		Hidden: true,
		Short:  "use `substrate account list`",
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
	}
}

func Main(context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {
	ui.Print("`substrate accounts` has been replaced by `substrate account list`; please run that command from now on")
	os.Exit(1)
}
