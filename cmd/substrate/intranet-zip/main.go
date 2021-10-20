package intranetzip

import (
	"os"

	createadminaccount "github.com/src-bin/substrate/cmd/substrate/create-admin-account"
)

func Main() {
	os.Stdout.Write(createadminaccount.SubstrateIntranetZip)
}
