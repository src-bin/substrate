package setup

import (
	"strings"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

func substrateAccountQuality(substrateAccount *awscfg.Account) (quality string) {
	quality = substrateAccount.Tags[tagging.Quality]
	if quality == "" {
		qualities, err := naming.Qualities()
		ui.Must(err)
		quality = qualities[0]
		if len(qualities) > 1 {
			ui.Printf(
				"found multiple qualities %s; choosing %s for your Substrate account (this is temporary and inconsequential)",
				strings.Join(qualities, ", "),
				quality,
			)
		}
	}
	return
}
