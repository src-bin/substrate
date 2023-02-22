package roles

import (
	"context"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsutil"
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
)

func TestCreateAndDeleteHumanRole(t *testing.T) {
	const (
		roleName    = "Foo"
		domain      = "foo"
		environment = "staging"
		quality     = "default"
	)
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	fooCfg := awscfg.Must(cfg.AssumeServiceRole(
		ctx,
		domain, environment, quality,
		roles.Administrator,
		time.Hour,
	))

	//testRoleDoesNotExist(t, ctx, cfg, roleName)
	//testRoleDoesNotExist(t, ctx, fooCfg, roleName)

	cmdutil.OverrideArgs("-domain", domain, "-humans", "-role", roleName)
	createrole.Main(ctx, cfg)

	testRoleExists(t, ctx, cfg, roleName)    // because -humans
	testRoleExists(t, ctx, fooCfg, roleName) // because -domain
	// TODO test that this role does not exist in any other accounts

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRoleDoesNotExist(t, ctx, cfg, roleName)
	testRoleDoesNotExist(t, ctx, fooCfg, roleName)
}

func TestCreateAndDeleteManagementRole(t *testing.T) {
	const roleName = "Mgmt"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.OrganizationAdministrator,
		time.Hour,
	))

	//testRoleDoesNotExist(t, ctx, mgmtCfg, roleName)

	cmdutil.OverrideArgs("-management", "-role", roleName)
	createrole.Main(ctx, cfg)

	testRoleDoesNotExist(t, ctx, cfg, roleName) // because no -humans
	testRoleExists(t, ctx, mgmtCfg, roleName)   // because -management
	// TODO test that this role does not exist in any other accounts

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRoleDoesNotExist(t, ctx, mgmtCfg, roleName)
}

func TestCreateAndDeleteSpecialRole(t *testing.T) {
	const roleName = "Special"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	deployCfg := awscfg.Must(cfg.AssumeSpecialRole(
		ctx,
		naming.Deploy,
		roles.DeployAdministrator,
		time.Hour,
	))
	networkCfg := awscfg.Must(cfg.AssumeSpecialRole(
		ctx,
		naming.Network,
		roles.NetworkAdministrator,
		time.Hour,
	))

	//testRoleDoesNotExist(t, ctx, deployCfg, roleName)
	//testRoleDoesNotExist(t, ctx, networkCfg, roleName)

	cmdutil.OverrideArgs("-role", roleName, "-special", naming.Deploy, "-special", naming.Network)
	createrole.Main(ctx, cfg)

	testRoleDoesNotExist(t, ctx, cfg, roleName)  // because no -humans
	testRoleExists(t, ctx, deployCfg, roleName)  // because -special deploy
	testRoleExists(t, ctx, networkCfg, roleName) // because -special network
	// TODO test that this role does not exist in any other accounts

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRoleDoesNotExist(t, ctx, deployCfg, roleName)
	testRoleDoesNotExist(t, ctx, networkCfg, roleName)
}

func testRoleDoesNotExist(t *testing.T, ctx context.Context, cfg *awscfg.Config, roleName string) {
	role, err := awsiam.GetRole(ctx, cfg, roleName)
	if err == nil {
		t.Fatalf("found %+v but expected NoSuchEntity", role)
	}
	if !awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
		t.Fatalf("error is %v but expected NoSuchEntity", err)
	}
	t.Log(role)
}

func testRoleExists(t *testing.T, ctx context.Context, cfg *awscfg.Config, roleName string) {
	role, err := awsiam.GetRole(ctx, cfg, roleName)
	if err != nil {
		t.Fatalf("error is %v but expected nil", err)
	}
	t.Log(role)
}
