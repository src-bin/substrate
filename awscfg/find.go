package awscfg

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tags"
)

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
		return a.Tags[tags.Domain] == domain && a.Tags[tags.Environment] == environment && a.Tags[tags.Quality] == quality
	})
}

// FindSpecialAccount returns the *Account for the admin account with the given
// name. It must be called in the management account.
func (c *Config) FindSpecialAccount(ctx context.Context, name string) (*Account, error) {
	return c.FindAccount(ctx, func(a *Account) bool {
		return a.Tags[tags.Name] == name
	})
}
