package accounts

import (
	"context"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/roles"
)

func TestSelectionAllService(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
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
	assertSlice(t, accountIds, []string{"306228446141", "509660714689", "765683503745", "832411437665"})
}

func TestSelectionFoo(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	selection := Selection{
		Domains:         []string{"foo"},
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
	assertSlice(t, accountIds, []string{"765683503745"})
}

func TestSelectionHumans(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	selection := Selection{Humans: true}
	selected, _, err := selection.Partition(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assert(t, len(selected), 1)
	assert(t, aws.ToString(selected[0].Account.Id), "716893237583")
}

func TestSelectionSubstrate(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	selection := Selection{Substrate: true}
	selected, _, err := selection.Partition(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	assert(t, len(selected), 1)
	assert(t, aws.ToString(selected[0].Account.Id), "716893237583")
}

func TestSelectionZero(t *testing.T) {
	s := &Selection{}
	if len(s.Arguments()) != 0 {
		t.Errorf("len(s.Arguments()): %d != 0; s.Arguments(): %+v", len(s.Arguments()), s.Arguments())
	}
	if s.String() != "" {
		t.Errorf(`s.String(): %q != ""`, s.String())
	}
}

func assert[T comparable](t *testing.T, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Errorf("actual: %+v != expected: %+v", actual, expected)
	}
}

func assertSlice[T comparable](t *testing.T, actual, expected []T) {
	t.Helper()
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
