package intranetzip

import (
	"os"

	"github.com/src-bin/substrate/awscfg"
	createadminaccount "github.com/src-bin/substrate/cmd/substrate/create-admin-account"
)

func Main(*awscfg.Main) {
	os.Stdout.Write(createadminaccount.SubstrateIntranetZip)
}
