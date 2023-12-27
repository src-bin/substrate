package createrole

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
		Use:    "create-role (deprecated)",
		Hidden: true,
		Short:  "TODO createrole.Command().Short",
		Long:   `TODO createrole.Command().Long`,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
	}
}

func Main(context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {
	ui.Print("`substrate create-role` has been replaced by `substrate role create`; please run that command from now on")
	os.Exit(1)
}
