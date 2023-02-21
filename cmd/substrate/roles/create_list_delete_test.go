package roles

import (
	"context"
	"testing"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsutil"
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
)

func TestCreateAndDeleteRole(t *testing.T) {
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	const roleName = "Foo"

	ensureRoleDoesNotExist(t, ctx, cfg, roleName)

	cmdutil.OverrideArgs("-domain", "foo", "-humans", "-role", roleName)
	createrole.Main(ctx, cfg)

	ensureRoleExists(t, ctx, cfg, roleName)

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	ensureRoleDoesNotExist(t, ctx, cfg, roleName)
}

func ensureRoleDoesNotExist(t *testing.T, ctx context.Context, cfg *awscfg.Config, roleName string) {
	role, err := awsiam.GetRole(ctx, cfg, roleName)
	if err == nil {
		t.Fatalf("found %+v but expected NoSuchEntity", role)
	}
	if !awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
		t.Fatalf("error is %v but expected NoSuchEntity", err)
	}
	t.Log(role)
}

func ensureRoleExists(t *testing.T, ctx context.Context, cfg *awscfg.Config, roleName string) {
	role, err := awsiam.GetRole(ctx, cfg, roleName)
	if err != nil {
		t.Fatalf("error is %v but expected nil", err)
	}
	t.Log(role)
}
