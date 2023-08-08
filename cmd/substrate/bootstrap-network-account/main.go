package bootstrapnetworkaccount

import (
	"context"
	"io"
	"os"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/ui"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	ui.Print("`substrate bootstrap-network-account` has been replaced by `substrate setup`; please run that command from now on")
	os.Exit(1)
}
