package awscfg

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

const (
	AccountsFilename       = "substrate.accounts.txt"
	CachedAccountsFilename = ".substrate.accounts.json" // cached on disk (obviously)
	MemoizedAccountsTTL    = time.Hour                  // memoized in memory
)

func (c *Config) ClearCachedAccounts() error {
	c.accounts = nil
	c.accountsExpiry = time.Time{}
	pathname, err := fileutil.PathnameInParents(CachedAccountsFilename)
	if err != nil && err != fs.ErrNotExist {
		return err
	}
	return fileutil.Remove(pathname)
}

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
// function returns true. It must be called in the management account. We
// expect to find an account that matches so if we don't we remove the cache
// file and try once more
func (c *Config) FindAccounts(
	ctx context.Context,
	f func(*Account) bool,
) (accounts []*Account, err error) {
	for i := 0; i < 2; i++ {

		// List all accounts, possibly from an on-disk or even in-memory cache,
		// and collect all the ones that pass the acceptance test function.
		var allAccounts []*Account
		if allAccounts, err = c.ListAccounts(ctx); err != nil {
			return
		}
		for _, account := range allAccounts {
			if f(account) {
				accounts = append(accounts, account)
			}
		}

		// If we've found any accounts, we're done. There's an outside
		// possibility that this result is incomplete. There are precious few
		// applications that even might return multiple accounts and the only
		// way one could be missing from ListAccounts is if it was just created
		// but we clear the cache before creating any accounts. So this seems
		// safe enough.
		if len(accounts) > 0 {
			break
		}

		// If we're not found any accounts, we need to clear the cache so that
		// the second pass we're about to do actually means something.
		if err = c.ClearCachedAccounts(); err != nil {
			return
		}

	}
	return
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

	// Return early if we have memoized accounts that are still valid.
	if c.accounts != nil && c.accountsExpiry.After(time.Now()) {
		return c.accounts, nil
	}

	// Return early if we have cached accounts.
	if pathname, err := fileutil.PathnameInParents(CachedAccountsFilename); err == nil {
		if err := jsonutil.Read(pathname, &accounts); err == nil {
			return accounts, err
		}
	}

	cfg, err := c.OrganizationReader(ctx)
	if err != nil {
		return nil, err
	}
	client := cfg.Organizations()

	// List all the accounts in the organization, even across multiple pages.
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

	// List the tags for every account in parallel.
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

	// Cache the full ListAccounts and ListTagsForResource amalgamation. Do not
	// treat an error here as fatal, since all it would do is slow us down.
	if err := jsonutil.Write(accounts, CachedAccountsFilename); err != nil {
		ui.Print(err)
	}

	// Memoize - cache in memory - the accounts, too.
	c.accounts = accounts
	c.accountsExpiry = time.Now().Add(MemoizedAccountsTTL)

	return
}
