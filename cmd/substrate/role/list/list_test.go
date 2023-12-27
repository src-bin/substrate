package list

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
)

func TestEC2(t *testing.T) {
	const roleName = "TestEC2"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	stdout := &strings.Builder{}

	testRole(t, ctx, cfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-special", naming.Deploy,
		"-humans", // TODO what's a better test, with this or without it?
		"-aws-service", "ec2.amazonaws.com",
	)
	createrole.Main(ctx, cfg, os.Stdout)

	cmdutil.OverrideArgs("-format", "json")
	Main(ctx, cfg, stdout)

	actual := stdout.String()
	expected := `[
	{
		"RoleName": "TestEC2",
		"AccountSelection": {
			"AllDomains": false,
			"Domains": null,
			"AllEnvironments": false,
			"Environments": null,
			"AllQualities": false,
			"Qualities": null,
			"Substrate": false,
			"Admin": false,
			"Management": false,
			"Specials": [
				"deploy"
			],
			"Numbers": null
		},
		"AssumeRolePolicy": {
			"Humans": true,
			"AWSServices": [
				"ec2.amazonaws.com"
			],
			"GitHubActions": null,
			"Filenames": null
		},
		"PolicyAttachments": {
			"AdministratorAccess": false,
			"ReadOnlyAccess": false,
			"ARNs": null,
			"Filenames": null
		},
		"RoleARNs": [
			"arn:aws:iam::716893237583:role/TestEC2",
			"arn:aws:iam::903998760555:role/TestEC2"
		]
	}
]
`
	if actual != expected {
		t.Errorf("`substrate roles -format json` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	stdout.Reset()

	cmdutil.OverrideArgs("-format", "shell")
	Main(ctx, cfg, stdout)

	actual = stdout.String()
	expected = `set -e -x
substrate create-role -role "TestEC2" -special "deploy" -humans -aws-service "ec2.amazonaws.com"
`
	if actual != expected {
		t.Errorf("`substrate roles -format shell` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, cfg, roleName, testNotExists)
}

func TestEverything(t *testing.T) {
	const roleName = "TestEverything"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	stdout := &strings.Builder{}

	testRole(t, ctx, cfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-domain", "foo",
		"-domain", "bar",
		"-environment", "staging",
		"-substrate",
		"-management",
		"-special", "deploy",
		"-special", "network",
		"-humans",
		"-aws-service", "ecs.amazonaws.com",
		"-aws-service", "lambda.amazonaws.com",
		"-github-actions", "src-bin/src-bin",
		"-github-actions", "src-bin/substrate",
		"-assume-role-policy", "policies/TestEverything.assume-role-policy.json",
		"-administrator-access",
		"-policy-arn", "arn:aws:iam::aws:policy/job-function/Billing",
		"-policy", "policies/TestEverything.policy.json",
	)
	createrole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, awscfg.Must(cfg.AssumeServiceRole(
		ctx,
		"baz", "staging", "default",
		roles.Administrator,
		time.Hour,
	)), roleName, testNotExists) // because no -domain "baz"

	cmdutil.OverrideArgs("-format", "json")
	Main(ctx, cfg, stdout)

	actual := stdout.String()
	expected := `[
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
			"Substrate": true,
			"Admin": false,
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
				"ecs.amazonaws.com",
				"lambda.amazonaws.com"
			],
			"GitHubActions": [
				"src-bin/src-bin",
				"src-bin/substrate"
			],
			"Filenames": [
				"policies/TestEverything.assume-role-policy.json"
			]
		},
		"PolicyAttachments": {
			"AdministratorAccess": true,
			"ReadOnlyAccess": false,
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
`
	if actual != expected {
		t.Errorf("`substrate roles -format json` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	stdout.Reset()

	cmdutil.OverrideArgs("-format", "shell")
	Main(ctx, cfg, stdout)

	actual = stdout.String()
	expected = `set -e -x
substrate create-role -role "TestEverything" -domain "bar" -domain "foo" -environment "staging" -all-qualities -substrate -management -special "deploy" -special "network" -humans -aws-service "ecs.amazonaws.com" -aws-service "lambda.amazonaws.com" -github-actions "src-bin/src-bin" -github-actions "src-bin/substrate" -assume-role-policy "policies/TestEverything.assume-role-policy.json" -administrator-access -policy-arn "arn:aws:iam::aws:policy/job-function/Billing" -policy "policies/TestEverything.policy.json"
`
	if actual != expected {
		t.Errorf("`substrate roles -format shell` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, cfg, roleName, testNotExists)
}

func TestZero(t *testing.T) {
	const roleName = "TestZero"
	defer cmdutil.RestoreArgs()
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	stdout := &strings.Builder{}

	testRole(t, ctx, cfg, roleName, testNotExists)

	cmdutil.OverrideArgs(
		"-role", roleName,
		"-special", naming.Deploy,
		"-aws-service", "sts.amazonaws.com", // dummy assume-role policy flag
	)
	createrole.Main(ctx, cfg, os.Stdout)

	cmdutil.OverrideArgs("-format", "json")
	Main(ctx, cfg, stdout)

	actual := stdout.String()
	expected := `[
	{
		"RoleName": "TestZero",
		"AccountSelection": {
			"AllDomains": false,
			"Domains": null,
			"AllEnvironments": false,
			"Environments": null,
			"AllQualities": false,
			"Qualities": null,
			"Substrate": false,
			"Admin": false,
			"Management": false,
			"Specials": [
				"deploy"
			],
			"Numbers": null
		},
		"AssumeRolePolicy": {
			"Humans": false,
			"AWSServices": [
				"sts.amazonaws.com"
			],
			"GitHubActions": null,
			"Filenames": null
		},
		"PolicyAttachments": {
			"AdministratorAccess": false,
			"ReadOnlyAccess": false,
			"ARNs": null,
			"Filenames": null
		},
		"RoleARNs": [
			"arn:aws:iam::903998760555:role/TestZero"
		]
	}
]
`
	if actual != expected {
		t.Errorf("`substrate roles -format json` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	stdout.Reset()

	cmdutil.OverrideArgs("-format", "shell")
	Main(ctx, cfg, stdout)

	actual = stdout.String()
	expected = `set -e -x
substrate create-role -role "TestZero" -special "deploy" -aws-service "sts.amazonaws.com"
`
	if actual != expected {
		t.Errorf("`substrate roles -format shell` output is wrong\nactual: %s\nexpected: %s", actual, expected) // TODO pass actual and expected to diff(1)
	}

	cmdutil.OverrideArgs("-delete", "-role", roleName)
	deleterole.Main(ctx, cfg, os.Stdout)

	testRole(t, ctx, cfg, roleName, testNotExists)
}