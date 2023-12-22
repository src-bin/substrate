package cmdutil

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
)

// Main returns the arguments necessary for a typical subcommand's Main
// function so that it can be called as Main(awscfg.Main(cmd.Context())).
func Main(cmd *cobra.Command, args []string) (context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {
	ctx := cmd.Context()
	return ctx, awscfg.Must(awscfg.NewConfig(ctx)), cmd, args, os.Stdout
}
