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
	substrateCfg := testawscfg.Test4(roles.Administrator)
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	sso(ctx, mgmtCfg)
}

func TestSSOTest8(t *testing.T) {
	ctx := context.Background()
	substrateCfg := testawscfg.Test8(roles.Administrator)
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	sso(ctx, mgmtCfg)
}
