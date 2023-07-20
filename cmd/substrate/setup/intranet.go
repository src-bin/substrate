package setup

import (
	"context"

	"github.com/src-bin/substrate/awscfg"
)

// intranet configures the Intranet in the Substrate account and returns the
// DNS domain name where it's being served. This is, at present, mostly a crib
// from `substrate create-admin-account`.
func intranet(ctx context.Context, cfg *awscfg.Config) string {
	return "example.com" // XXX this won't work XXX
}
