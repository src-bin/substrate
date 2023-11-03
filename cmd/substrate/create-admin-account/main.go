package createadminaccount

import (
	"context"
	"io"
	"os"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/ui"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	ui.Print("`substrate create-admin-account` has been replaced by `substrate setup`; please run that command from now on")
	os.Exit(1)
}

// Synopsis returns a one-line, short synopsis of the command.
func Synopsis() string {
	return "deprecated: use `substrate setup`"
}
