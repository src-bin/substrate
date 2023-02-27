package roles

import (
	"context"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
)

func TestEverything(t *testing.T) {
	const roleName = "Everything"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)

	testRole(t, ctx, cfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-domain", "foo",
		"-domain", "bar",
		"-environment", "staging",
		"-management",
		"-special", "deploy",
		"-special", "network",
		"-humans",
		"-aws-service", "ec2.amazonaws.com",
		"-aws-service", "lambda.amazonaws.com",
		"-github-actions", "src-bin/src-bin",
		"-github-actions", "src-bin/substrate",
		"-assume-role-policy", "policies/FooBar.assume-role-policy.json",
		"-administrator",
		"-policy-arn", "arn:aws:iam::aws:policy/job-function/Billing",
		"-policy", "policies/FooBar.policy.json",
	)
	createrole.Main(ctx, cfg)

	testRole(t, ctx, awscfg.Must(cfg.AssumeServiceRole(
		ctx,
		"baz", "staging", "default",
		roles.Administrator,
		time.Hour,
	)), roleName, testNotExists) // because no -domain "baz"

	cmdutil.OverrideArgs("-format", "json")
	Main(ctx, cfg)
	// TODO figure out how to make assertions on its output

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)
}
