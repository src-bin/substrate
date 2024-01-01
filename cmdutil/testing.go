package cmdutil

import (
	"os"

	"github.com/spf13/cobra"
)

// OverrideArgs provides a way to test main functions that use package-level
// flag functions to parse command-line arguments. It must be called _before_
// calling and package-level flag functions.
func OverrideArgs(cmd *cobra.Command, args ...string) {
	cmd.Flags().Parse(args)
}

// RestoreArgs puts everything back where it was. It should be called in a
// defer statement at the beginning of a test or benchmark function.
func RestoreArgs() {
	os.Args = append([]string{}, originalArgs...)
}

func init() {
	originalArgs = append([]string{}, os.Args...)
}

var originalArgs []string
