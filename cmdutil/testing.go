package cmdutil

import (
	"flag"
	"os"
)

func OverrideArgs(args ...string) {
	os.Args = append(os.Args[0:1], args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func RestoreArgs() {
	os.Args = append(os.Args[0:1], originalArgs...)
}

func init() {
	originalArgs = append(originalArgs, os.Args[1:]...)
}

var originalArgs []string
