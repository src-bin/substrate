package awscfg

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
)

const MemoizedAccountsTTL = time.Hour

// FindAccount returns the first *Account for which the given acceptance test
// function returns true. It must be called in the management account.
func (c *Config) FindAccount(
	ctx context.Context,
	f func(*Account) bool,
) (*Account, error) {
	accounts, err := c.FindAccounts(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, nil
	}
	if len(accounts) > 1 {
		return nil, fmt.Errorf("found %d AWS accounts when expecting only one", len(accounts))
	}
	return accounts[0], nil
}

// FindAccounts returns all []*Account for which the given acceptance test
// function returns true. It must be called in the management account.
// TODO memoize this function by unifying it with awsorgs.ListAccounts.
func (c *Config) FindAccounts(
	ctx context.Context,
	f func(*Account) bool,
) ([]*Account, error) {
	cfg, err := c.OrganizationReader(ctx)
	if err != nil {
		return nil, err
	}
	client := cfg.Organizations()

	var (
		accounts  []*Account
		nextToken *string
	)
	for {
		out, err := client.ListAccounts(ctx, &organizations.ListAccountsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, a := range out.Accounts {
			account := &Account{Account: a}
			account.Tags, err = cfg.listTagsForResource(ctx, aws.ToString(account.Id))
			if err != nil {
				return nil, err
			}
			if f(account) {
				accounts = append(accounts, account)
			}
		}

		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return accounts, nil
}

// FindAdminAccount returns the *Account for the admin account with the given
// quality. It must be called in the management account.
func (c *Config) FindAdminAccount(ctx context.Context, quality string) (*Account, error) {
	return c.FindServiceAccount(ctx, naming.Admin, naming.Admin, quality)
}

// FindServiceAccount returns the *Account for the admin account with the
// given domain, environment, and quality. It must be called in the
// management account.
func (c *Config) FindServiceAccount(ctx context.Context, domain, environment, quality string) (*Account, error) {
	return c.FindAccount(ctx, func(a *Account) bool {
		return a.Tags[tagging.Domain] == domain && a.Tags[tagging.Environment] == environment && a.Tags[tagging.Quality] == quality
	})
}

// FindSpecialAccount returns the *Account for the admin account with the given
// name. It must be called in the management account.
func (c *Config) FindSpecialAccount(ctx context.Context, name string) (*Account, error) {
	return c.FindAccount(ctx, func(a *Account) bool {
		return a.Tags[tagging.Name] == name
	})
}

func (c *Config) ListAccounts(ctx context.Context) (accounts []*Account, err error) {
	defer func(t0 time.Time) { log.Print(time.Since(t0)) }(time.Now())
	if c.accounts != nil && c.accountsExpiry.After(time.Now()) {
		return c.accounts, nil
	}

	cfg, err := c.OrganizationReader(ctx)
	if err != nil {
		return nil, err
	}
	client := cfg.Organizations()

	var nextToken *string
	for {
		out, err := client.ListAccounts(ctx, &organizations.ListAccountsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, account := range out.Accounts {

			// Don't return suspended (read: closed, deleted) accounts here as
			// there's really nothing we can do with them except cause problems
			// downstream.
			if account.Status == types.AccountStatusSuspended {
				continue
			}

			accounts = append(accounts, &Account{Account: account})
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}

	ch := make(chan error, len(accounts))
	for i := 0; i < len(accounts); i++ {
		go func(i int) {
			var err error
			accounts[i].Tags, err = cfg.listTagsForResource(ctx, aws.ToString(accounts[i].Id))
			ch <- err
		}(i) // pass i so goroutines don't all refer to the same address
	}
	for i := 0; i < len(accounts); i++ {
		if err = <-ch; err != nil {
			return nil, err
		}
	}

	c.accounts = accounts
	c.accountsExpiry = time.Now().Add(MemoizedAccountsTTL)
	return
}
