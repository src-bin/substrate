package intranetzip

import (
	"context"
	"flag"
	"os"

	"github.com/src-bin/substrate/awscfg"
	createadminaccount "github.com/src-bin/substrate/cmd/substrate/create-admin-account"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	flag.Usage = func() {
		ui.Print("Usage: substrate intranet-zip")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Though this command doesn't create any AWS accounts or apply any
	// Terraform modules, which are the usual things we're eager to prevent
	// happening based on old code, it would be bad to downgrade a customer's
	// Intranet and this is an effective way to prevent that.
	versionutil.PreventDowngrade(ctx, cfg)

	os.Stdout.Write(createadminaccount.SubstrateIntranetZip)
}
