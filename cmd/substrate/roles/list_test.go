package roles

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/roles"
)

func TestEverything(t *testing.T) {
	const roleName = "TestEverything"
	defer cmdutil.RestoreArgs()
	ctx := stdoutContext(t, "TestEverything-*.stdout")
	pathname := contextutil.ValueString(ctx, contextutil.RedirectStdoutTo)
	defer os.Remove(pathname)
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
		"-assume-role-policy", "policies/TestEverything.assume-role-policy.json",
		"-administrator",
		"-policy-arn", "arn:aws:iam::aws:policy/job-function/Billing",
		"-policy", "policies/TestEverything.policy.json",
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

	actual, err := fileutil.ReadFile(pathname)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte(`[
	{
		"RoleName": "TestEverything",
		"AccountSelection": {
			"AllDomains": false,
			"Domains": [
				"bar",
				"foo"
			],
			"AllEnvironments": false,
			"Environments": [
				"staging"
			],
			"AllQualities": true,
			"Qualities": null,
			"Admin": true,
			"Management": true,
			"Specials": [
				"deploy",
				"network"
			],
			"Numbers": null
		},
		"AssumeRolePolicy": {
			"Humans": true,
			"AWSServices": [
				"ec2.amazonaws.com",
				"lambda.amazonaws.com"
			],
			"GitHubActions": null,
			"Filenames": [
				"policies/TestEverything.assume-role-policy.json"
			]
		},
		"PolicyAttachments": {
			"Administrator": true,
			"ReadOnly": false,
			"ARNs": [
				"arn:aws:iam::aws:policy/job-function/Billing"
			],
			"Filenames": [
				"policies/TestEverything.policy.json"
			]
		},
		"RoleARNs": [
			"arn:aws:iam::306228446141:role/TestEverything",
			"arn:aws:iam::488246444926:role/TestEverything",
			"arn:aws:iam::617136454425:role/TestEverything",
			"arn:aws:iam::716893237583:role/TestEverything",
			"arn:aws:iam::765683503745:role/TestEverything",
			"arn:aws:iam::903998760555:role/TestEverything"
		]
	}
]
`)
	if !bytes.Equal(actual, expected) {
		t.Errorf("`substrate roles -format json` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	if err := os.Truncate(pathname, 0); err != nil {
		t.Fatal(err)
	}

	cmdutil.OverrideArgs("-format", "shell")
	Main(ctx, cfg)

	actual, err = fileutil.ReadFile(pathname)
	if err != nil {
		t.Fatal(err)
	}
	expected = []byte(`set -e -x
substrate create-role -role "TestEverything" -domain "bar" -domain "foo" -environment "staging" -all-qualities -admin -management -special "deploy" -special "network" -humans -aws-service "ec2.amazonaws.com" -aws-service "lambda.amazonaws.com" -assume-role-policy "policies/TestEverything.assume-role-policy.json" -administrator -policy-arn "arn:aws:iam::aws:policy/job-function/Billing" -policy "policies/TestEverything.policy.json"
`)
	if !bytes.Equal(actual, expected) {
		t.Errorf("`substrate roles -format shell` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg)

	testRole(t, ctx, cfg, roleName, testNotExists)
}

func stdoutContext(t *testing.T, pattern string) context.Context {
	t.Helper()
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		t.Fatal(err)
		return context.Background()
	}
	pathname := f.Name()
	if err := f.Close(); err != nil {
		t.Fatal(err)
		return context.Background()
	}
	return context.WithValue(
		context.Background(),
		contextutil.RedirectStdoutTo,
		pathname,
	)
}
