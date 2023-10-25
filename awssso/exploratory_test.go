package awssso

import (
	"context"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/roles"
)

func TestListInstancesTest4(t *testing.T) {
	ctx := context.Background()
	substrateCfg := testawscfg.Test4(roles.Administrator)
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	if err := awsorgs.RegisterDelegatedAdministrator(
		ctx,
		mgmtCfg,
		substrateCfg.MustAccountId(ctx),
		"sso.amazonaws.com",
	); err != nil {
		t.Fatal(err)
	}

	testListInstances(t, ctx, mgmtCfg)
	// I never bothered to setup us-east-2 when I was doing it manually but
	// I did set us-west-2 up. Nice regional SPOF you've got there, Richard.
}

func TestListInstancesTest8(t *testing.T) {
	ctx := context.Background()
	substrateCfg := testawscfg.Test8(roles.Administrator)
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	testListInstances(t, ctx, mgmtCfg)
}

func testListInstances(
	t *testing.T,
	ctx context.Context,
	mgmtCfg *awscfg.Config,
) {
	instances, err := ListInstances(ctx, mgmtCfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(jsonutil.MustString(instances))
}
