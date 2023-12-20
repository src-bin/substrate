package awscfg

import (
	"context"
	"io"
	"os"
)

// Main returns the arguments necessary for a typical subcommand's Main
// function so that it can be called as Main(awscfg.Main(cmd.Context())).
func Main(ctx context.Context) (context.Context, *Config, io.Writer) {
	return ctx, Must(NewConfig(ctx)), os.Stdout
}
