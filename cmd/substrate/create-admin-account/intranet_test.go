package createadminaccount

import (
	"context"
	"testing"

	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/roles"
)

func TestIntranet(t *testing.T) {
	t.Skip()
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	if err := ensureIntranet(
		ctx,
		cfg,
		"src-bin-test1.net",
		"client-id",     // XXX
		"client-secret", // XXX
		"dev-662445.okta.com",
		"", // unused Azure AD tenant ID
	); err != nil {
		t.Fatal(err)
	}
	// TODO actually test that this Intranet works
}
