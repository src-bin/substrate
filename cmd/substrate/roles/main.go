package roles

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
		Use:    "roles (deprecated)",
		Hidden: true,
		Short:  "TODO roles.Command().Short",
		Long:   `TODO roles.Command().Long`,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
	}
}

func Main(context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {
	ui.Print("`substrate roles` has been replaced by `substrate role list`; please run that command from now on")
	os.Exit(1)
}
