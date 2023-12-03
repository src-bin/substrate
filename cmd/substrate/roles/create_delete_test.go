package roles

import (
	"context"
	"os"
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
)

func TestFooBarBaz(t *testing.T) {
	const roleName = "TestFooBarBaz"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	_, serviceAccounts, _, _, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-all-domains",
		"-all-environments",
		// "-all-qualities", // test that we can omit this with only one quality as we have in test1
		"-aws-service", "sts.amazonaws.com", // dummy assume-role policy flag
	)
	createrole.Main(ctx, cfg, os.Stdout)

	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testExists) // because -all-{domains,environments,qualities}
	testRole(t, ctx, cfg, roleName, testNotExists)                         // because no -admin or -humans
	testRoleInAccounts(t, ctx, cfg, []*awsorgs.Account{
		managementAccount, // because no -management
		deployAccount,     // because no -special "deploy"
		networkAccount,    // because no -special "network"
	}, roleName, testNotExists)

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, cfg, roleName, testNotExists)
	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists)
}

func TestFooHumans(t *testing.T) {
	const (
		roleName    = "TestFoo"
		domain      = "foo"
		otherDomain = "bar"
		environment = "staging"
		quality     = "default"
	)
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	fooCfg := awscfg.Must(cfg.AssumeServiceRole(
		ctx,
		domain, environment, quality,
		roles.Administrator,
		time.Hour,
	))

	testRole(t, ctx, cfg, roleName, testNotExists)
	testRole(t, ctx, fooCfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-domain", domain,
		"-all-environments",
		"-all-qualities",
		"-humans",
	)
	createrole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, fooCfg, roleName, testExists) // because -domain "foo"
	testRole(t, ctx, cfg, roleName, testExists)    // because -humans
	testRole(t, ctx, awscfg.Must(cfg.AssumeServiceRole(
		ctx,
		otherDomain, environment, quality,
		roles.Administrator,
		time.Hour,
	)), roleName, testNotExists) // because no -all-domains or -domain "bar"

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, cfg, roleName, testNotExists)
	testRole(t, ctx, fooCfg, roleName, testNotExists)
}

func TestManagement(t *testing.T) {
	const roleName = "TestMgmt"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	_, serviceAccounts, _, _, deployAccount, _, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	testRole(t, ctx, mgmtCfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-management",
		"-aws-service", "sts.amazonaws.com", // dummy assume-role policy flag
	)
	createrole.Main(ctx, cfg, os.Stdout)

	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists) // because no -domain, -environment, or -quality
	testRole(t, ctx, cfg, roleName, testNotExists)                            // because no -admin or -humans
	testRole(t, ctx, mgmtCfg, roleName, testExists)                           // because -management
	testRoleInAccounts(t, ctx, cfg, []*awsorgs.Account{
		deployAccount,  // because no -special "deploy"
		networkAccount, // because no -special "network"
	}, roleName, testNotExists)

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, mgmtCfg, roleName, testNotExists)
}

func TestSpecial(t *testing.T) {
	const roleName = "TestSpecial"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
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
	_, serviceAccounts, _, _, _, _, _, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	testRole(t, ctx, deployCfg, roleName, testNotExists)
	testRole(t, ctx, networkCfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-special", naming.Deploy,
		"-special", naming.Network,
		"-aws-service", "sts.amazonaws.com", // dummy assume-role policy flag
	)
	createrole.Main(ctx, cfg, os.Stdout)

	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists) // because no -all-*, -domain, -environment, or -quality
	testRole(t, ctx, cfg, roleName, testNotExists)                            // because no -admin or -humans
	testRole(t, ctx, deployCfg, roleName, testExists)                         // because -special deploy
	testRole(t, ctx, networkCfg, roleName, testExists)                        // because -special network

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, deployCfg, roleName, testNotExists)
	testRole(t, ctx, networkCfg, roleName, testNotExists)
}

func TestSubstrate(t *testing.T) {
	const roleName = "TestSubstrate"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	_, serviceAccounts, _, _, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	testRole(t, ctx, cfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-substrate",
		"-aws-service", "sts.amazonaws.com", // dummy assume-role policy flag
	)
	createrole.Main(ctx, cfg, os.Stdout)

	testRoleInAccounts(t, ctx, cfg, serviceAccounts, roleName, testNotExists) // because no -all-*, -domain, -environment, or -quality
	testRole(t, ctx, cfg, roleName, testExists)                               // because -substrate
	testRoleInAccounts(t, ctx, cfg, []*awsorgs.Account{
		managementAccount, // because no -management
		deployAccount,     // because no -special "deploy"
		networkAccount,    // because no -special "network"
	}, roleName, testNotExists)

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, cfg, roleName, testNotExists)
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
	t.Helper()
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
	t.Helper()
	for _, account := range accounts {
		testRole(t, ctx, awscfg.Must(cfg.AssumeRole(
			ctx,
			aws.ToString(account.Id),
			account.AdministratorRoleName(),
			time.Hour,
		)), roleName, test)
	}
}
