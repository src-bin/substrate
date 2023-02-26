package cmdutil

import (
	"flag"
	"os"
)

// OverrideArgs provides a way to test main functions that use package-level
// flag functions to parse command-line arguments. It must be called _before_
// calling and package-level flag functions.
func OverrideArgs(args ...string) {
	os.Args = append(os.Args[0:1], args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

// RestoreArgs puts everything back where it was. It should be called in a
// defer statement at the beginning of a test or benchmark function.
func RestoreArgs() {
	os.Args = append(os.Args[0:1], originalArgs...)
}

func init() {
	originalArgs = append(originalArgs, os.Args[1:]...)
}

var originalArgs []string
