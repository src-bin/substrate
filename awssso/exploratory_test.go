package awssso

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/roles"
)

func TestListInstancesTest4(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test4(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	if err := awsorgs.RegisterDelegatedAdministrator(
		ctx,
		mgmtCfg,
		substrateCfg.MustAccountId(ctx),
		"sso.amazonaws.com",
	); err != nil {
		t.Fatal(err)
	}

	accounts, err := mgmtCfg.ListAccounts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	instances := testListInstances(t, ctx, mgmtCfg)
	t.Log(jsonutil.MustString(instances))
	for _, instance := range instances {
		permissionSets, err := ListPermissionSets(ctx, mgmtCfg, instance)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(jsonutil.MustString(permissionSets))
		for _, permissionSet := range permissionSets {
			for _, account := range accounts {
				assignments, err := ListAccountAssignments(ctx, mgmtCfg, instance, permissionSet, aws.ToString(account.Id))
				if err != nil {
					t.Fatal(err)
				}
				t.Log(
					aws.ToString(instance.InstanceArn),
					aws.ToString(permissionSet.PermissionSetArn),
					aws.ToString(account.Id),
					jsonutil.MustString(assignments),
				)
			}
		}
	}
}

func TestListInstancesTest8(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test8(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))

	instances := testListInstances(t, ctx, mgmtCfg)
	t.Log(jsonutil.MustString(instances))
}

func testListInstances(
	t *testing.T,
	ctx context.Context,
	mgmtCfg *awscfg.Config,
) []*Instance {
	instances, err := ListInstances(ctx, mgmtCfg)
	if err != nil {
		t.Fatal(err)
	}
	return instances
}
