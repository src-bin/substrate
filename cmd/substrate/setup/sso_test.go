package setup

import (
	"context"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/roles"
)

func TestSSOTest4(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test4(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	sso(ctx, mgmtCfg)
}

func TestSSOTest8(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test8(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	sso(ctx, mgmtCfg)
}
