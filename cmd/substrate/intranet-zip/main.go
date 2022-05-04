package intranetzip

import (
	"context"
	"flag"
	"os"

	"github.com/src-bin/substrate/awscfg"
	createadminaccount "github.com/src-bin/substrate/cmd/substrate/create-admin-account"
	"github.com/src-bin/substrate/ui"
)

func Main(context.Context, *awscfg.Config) {
	flag.Usage = func() {
		ui.Print("Usage: substrate intranet-zip")
		flag.PrintDefaults()
	}
	flag.Parse()

	os.Stdout.Write(createadminaccount.SubstrateIntranetZip)
}
