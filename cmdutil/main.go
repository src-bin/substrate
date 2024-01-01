package cmdutil

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
)

// Main returns the arguments necessary for a typical subcommand's Main
// function so that it can be called as Main(cmdutil.Main(cmd.Context())).
func Main(cmd *cobra.Command, args []string) (context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {
	return MainRedirect(cmd, args, os.Stdout)
}

// MainRedirect returns the arguments necessary for a typical subcommand's
// Main function with its output io.Writer redirected to w. Call the Main
// function as Main(cmdutil.MainRedirect(cmd.Context(), w)).
func MainRedirect(cmd *cobra.Command, args []string, w io.Writer) (context.Context, *awscfg.Config, *cobra.Command, []string, io.Writer) {
	ctx := cmd.Context()
	return ctx, awscfg.Must(awscfg.NewConfig(ctx)), cmd, args, w
}
