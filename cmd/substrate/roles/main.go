package roles

import (
	"context"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/ui"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	ui.Fatal("not implemented")
	// TODO need a `substrate roles` command that lists Administrator, Auditor, and all the others folks define, complete with a `-format shell` variant that works like `substrate accounts -format shell`
}
