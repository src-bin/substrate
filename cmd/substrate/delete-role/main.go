package deleterole

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
		Use:    "delete-role (deprecated)",
		Hidden: true,
		Short:  "TODO deleterole.Command().Short",
		Long:   `TODO deleterole.Command().Long`,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
	}
}

func Main(context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {
	ui.Print("`substrate delete-role` has been replaced by `substrate role delete`; please run that command from now on")
	os.Exit(1)
}
