package roles

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsutil"
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

func TestCreateAndDeleteHumanRole(t *testing.T) {
	const (
		roleName    = "Foo"
		domain      = "foo"
		otherDomain = "bar"
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

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-domain", domain,
		"-all-environments",
		"-all-qualities",
		"-humans",
	)
	createrole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testExists)    // because -humans
	testRole(t, ctx, fooCfg, roleName, testExists) // because -domain "foo"
	testRole(t, ctx, awscfg.Must(cfg.AssumeServiceRole(
		ctx,
		otherDomain, environment, quality,
		roles.Administrator,
		time.Hour,
	)), roleName, testNotExists) // because -domain "foo" not -domain "bar"

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
	_, serviceAccounts, _, deployAccount, _, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	//testRole(t, ctx, mgmtCfg, roleName, testNotExists)

	cmdutil.OverrideArgs("-role", roleName, "-management")
	createrole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)                            // because no -humans
	testRole(t, ctx, mgmtCfg, roleName, testExists)                           // because -management
	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists) // because no -domain, -environment, or -quality
	testRoleInAccounts(t, ctx, cfg, []*awsorgs.Account{
		deployAccount,  // because no -special "deploy"
		networkAccount, // because no -special "network"
	}, roleName, testNotExists)

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, mgmtCfg, roleName, testNotExists)
}

func TestCreateAndDeleteServiceRole(t *testing.T) {
	const roleName = "FooBarBaz"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	_, serviceAccounts, _, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-all-domains",
		"-all-environments",
		// "-all-qualities", // test that we can omit this with only one quality as we have in test1
	)
	createrole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)                         // because no -humans
	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testExists) // because -all-{domains,environments,qualities}
	testRoleInAccounts(t, ctx, cfg, []*awsorgs.Account{
		managementAccount, // because no -management
		deployAccount,     // because no -special "deploy"
		networkAccount,    // because no -special "network"
	}, roleName, testNotExists)

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)
	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists)
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
	_, serviceAccounts, _, _, _, _, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	//testRole(t, ctx, deployCfg, roleName, testNotExists)
	//testRole(t, ctx, networkCfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-special", naming.Deploy,
		"-special", naming.Network,
	)
	createrole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)                            // because no -humans
	testRole(t, ctx, deployCfg, roleName, testExists)                         // because -special deploy
	testRole(t, ctx, networkCfg, roleName, testExists)                        // because -special network
	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists) // because no -domain, -environment, or -quality

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, deployCfg, roleName, testNotExists)
	testRole(t, ctx, networkCfg, roleName, testNotExists)
}

func init() {
	ui.Quiet()
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
			t.Errorf("expected a role in account %s but got %v", cfg.MustAccountId(ctx), err)
		}
	} else {
		if err == nil {
			t.Errorf("expected NoSuchEntity in account %s but got %s", cfg.MustAccountId(ctx), role.ARN)
		} else if !awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
			t.Errorf("expected NoSuchEntity in account %s but got %v", cfg.MustAccountId(ctx), err)
		}
	}
	//t.Log(role)
}

func testRoleInAccounts(
	t *testing.T,
	ctx context.Context,
	cfg *awscfg.Config,
	accounts []*awsorgs.Account,
	roleName string,
	test bool,
) {
	for _, account := range accounts {
		testRole(t, ctx, awscfg.Must(cfg.AssumeRole(
			ctx,
			aws.ToString(account.Id),
			account.AdministratorRoleName(),
			time.Hour,
		)), roleName, test)
	}
}
