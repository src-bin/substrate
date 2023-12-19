package accounts

import (
	"context"
	"io"
	"os"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/ui"
)

func Main(context.Context, *awscfg.Config, io.Writer) {
	ui.Print("`substrate accounts` has been replaced by `substrate account list`; please run that command from now on")
	os.Exit(1)
}
