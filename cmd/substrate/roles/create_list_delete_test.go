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

	//testRole(t, ctx, cfg, roleName, testNotExists)
	//testRole(t, ctx, fooCfg, roleName, testNotExists)

	cmdutil.OverrideArgs("-domain", domain, "-humans", "-role", roleName)
	createrole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testExists)    // because -humans
	testRole(t, ctx, fooCfg, roleName, testExists) // because -domain
	// TODO test that this role does not exist in any other accounts

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)
	testRole(t, ctx, fooCfg, roleName, testNotExists)
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

	//testRole(t, ctx, mgmtCfg, roleName, testNotExists)

	cmdutil.OverrideArgs("-management", "-role", roleName)
	createrole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)  // because no -humans
	testRole(t, ctx, mgmtCfg, roleName, testExists) // because -management
	// TODO test that this role does not exist in any other accounts

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, mgmtCfg, roleName, testNotExists)
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

	//testRole(t, ctx, deployCfg, roleName, testNotExists)
	//testRole(t, ctx, networkCfg, roleName, testNotExists)

	cmdutil.OverrideArgs("-role", roleName, "-special", naming.Deploy, "-special", naming.Network)
	createrole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)     // because no -humans
	testRole(t, ctx, deployCfg, roleName, testExists)  // because -special deploy
	testRole(t, ctx, networkCfg, roleName, testExists) // because -special network
	// TODO test that this role does not exist in any other accounts

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, deployCfg, roleName, testNotExists)
	testRole(t, ctx, networkCfg, roleName, testNotExists)
}

const (
	testExists    = true
	testNotExists = false
)

func testRole(
	t *testing.T,
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	test bool,
) {
	role, err := awsiam.GetRole(ctx, cfg, roleName)
	if test == testExists {
		if err != nil {
			t.Fatalf("expected nil but got %v", err)
		}
	} else {
		if err == nil {
			t.Fatalf("expected NoSuchEntity but got %+v", role)
		}
		if !awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
			t.Fatalf("expected NoSuchEntity but got %v", err)
		}
	}
	t.Log(role)
}
