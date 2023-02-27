package accounts

import (
	"context"
	"flag"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
)

/*
	cmdutil.OverrideArgs(
		"-role", "FooBar",
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
*/

func TestSelectionAdmin(t *testing.T) {
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	selection := Selection{Admin: true}
	selected, _, err := selection.Partition(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assert(t, len(selected), 1)
	assert(t, aws.ToString(selected[0].Account.Id), "716893237583")
}

func TestSelectionAllService(t *testing.T) {
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	selection := Selection{
		AllDomains:      true,
		AllEnvironments: true,
		AllQualities:    true,
	}
	selected, _, err := selection.Partition(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var accountIds []string
	for _, as := range selected {
		accountIds = append(accountIds, aws.ToString(as.Account.Id))
	}
	sort.Strings(accountIds)
	assertSlice(t, accountIds, []string{"306228446141", "765683503745", "832411437665"})
}

func TestSelectionFlagsToSelection1(t *testing.T) {
	defer cmdutil.RestoreArgs()
	cmdutil.OverrideArgs(
		"-domain", "bar", // these must be in alphabetical order because
		"-domain", "foo", // they will be when they're regurgitated
		"-environment", "staging",
		"-management",
		"-special", "deploy", // same for
		"-special", "network", // these
	)
	f := NewSelectionFlags(SelectionFlagsUsage{"-", "-", "-", "-", "-", "-", "-", "-", "-", "-"})
	flag.Parse()
	s, err := f.Selection()
	if err != nil {
		t.Fatal(err)
	}
	assert(t, s.AllDomains, false)
	assertSlice(t, s.Domains, []string{"bar", "foo"})
	assert(t, s.AllEnvironments, false)
	assertSlice(t, s.Environments, []string{"staging"})
	assert(t, s.AllQualities, true) // there's only one quality in test1
	assertSlice(t, s.Qualities, []string{})
	assert(t, s.Admin, false)
	assert(t, s.Management, true)
	assertSlice(t, s.Specials, []string{"deploy", "network"})
	assertSlice(t, s.Numbers, []string{})
}

func TestSelectionFlagsToSelection2(t *testing.T) {
	f := &SelectionFlags{
		AllDomains:      aws.Bool(true),
		Domains:         &cmdutil.StringSliceFlag{},
		AllEnvironments: aws.Bool(true),
		Environments:    &cmdutil.StringSliceFlag{},
		AllQualities:    aws.Bool(false),
		Qualities:       &cmdutil.StringSliceFlag{"default"},
		Admin:           aws.Bool(true),
		Management:      aws.Bool(false),
		Specials:        &cmdutil.StringSliceFlag{},
		Numbers:         &cmdutil.StringSliceFlag{"123456789012"},
	}
	s, err := f.Selection()
	if err != nil {
		t.Fatal(err)
	}
	assert(t, s.AllDomains, *f.AllDomains)
	assertSlice(t, s.Domains, f.Domains.Slice())
	assert(t, s.AllEnvironments, *f.AllEnvironments)
	assertSlice(t, s.Environments, f.Environments.Slice())
	assert(t, s.AllQualities, *f.AllQualities)
	assertSlice(t, s.Qualities, f.Qualities.Slice())
	assert(t, s.Admin, *f.Admin)
	assert(t, s.Management, *f.Management)
	assertSlice(t, s.Specials, f.Specials.Slice())
	assertSlice(t, s.Numbers, f.Numbers.Slice())
}

func assert[T comparable](t *testing.T, actual, expected T) {
	if actual != expected {
		t.Errorf("actual: %+v != expected: %+v", actual, expected)
	}
}

func assertSlice[T comparable](t *testing.T, actual, expected []T) {
	if len(actual) != len(expected) {
		t.Errorf("len(actual): %d != len(expected): %d", len(actual), len(expected))
		return
	}
	if len(actual) == 0 {
		return
	}
	for i := 0; i < len(actual); i++ {
		assert(t, actual[i], expected[i])
	}
}
