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
	if err := awsorgs.RegisterDelegatedAdministrator(
		ctx,
		awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour)),
		substrateCfg.MustAccountId(ctx),
		"sso.amazonaws.com",
	); err != nil {
		t.Fatal(err)
	}
	testListInstances(t, ctx, substrateCfg, []string{
		"us-east-2", // I never bothered to setup this region when I was doing it manually
		"us-west-2", // but I did set this one up; nice regional SPOF you've got there, Richard
	})
}

func TestListInstancesTest8(t *testing.T) {
	testListInstances(t, context.Background(), testawscfg.Test8(roles.Administrator), []string{"us-west-2"})
}

func testListInstances(
	t *testing.T,
	ctx context.Context,
	substrateCfg *awscfg.Config,
	regions []string,
) {
	for _, region := range regions {
		instances, err := ListInstances(ctx, substrateCfg.Regional(region))
		if err != nil {
			t.Fatal(err)
		}
		t.Log(region, jsonutil.MustString(instances))
	}
}
