package awscfg

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
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
// function returns true. This may be called from any account.
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
// function returns true. This may be called from any account. We expect to
// find an account that matches so if we don't we remove the cache file and
// try once more.
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

		// If we've not found any accounts, we need to clear the cache so that
		// the second pass we're about to do actually means something.
		if err = c.ClearCachedAccounts(); err != nil {
			return
		}

	}
	return
}

// FindAdminAccount returns the *Account for the admin account with the given
// quality. This may be called from any account.
func (c *Config) FindAdminAccount(ctx context.Context, quality string) (*Account, error) {
	return c.FindServiceAccount(ctx, naming.Admin, naming.Admin, quality)
}

// FindAdminAccounts returns an *Account for every admin account. This will,
// in practice, never return more than one account, but it does so without
// the caller having to know the quality ahead of time. Thankfully, it's also
// deprecated the moment it's introduced since admin accounts are going away
// in favor of the singular and simplified Substrate account. This may be
// called from any account.
func (c *Config) FindAdminAccounts(ctx context.Context) ([]*Account, error) {
	return c.FindAccounts(ctx, func(a *Account) bool {
		return a.Tags[tagging.Domain] == naming.Admin
	})
}

// FindManagementAccount returns the *Account for the management account. This
// may be called from any account.
func (c *Config) FindManagementAccount(ctx context.Context) (*Account, error) {
	return c.FindAccount(ctx, func(a *Account) bool {
		return a.Tags[tagging.SubstrateSpecialAccount] == naming.Management || a.Tags[tagging.SubstrateType] == naming.Management
	})
}

// FindServiceAccount returns the *Account for the admin account with the
// given domain, environment, and quality. This may be called from any account.
func (c *Config) FindServiceAccount(ctx context.Context, domain, environment, quality string) (*Account, error) {
	return c.FindAccount(ctx, func(a *Account) bool {
		return a.Tags[tagging.Domain] == domain && a.Tags[tagging.Environment] == environment && a.Tags[tagging.Quality] == quality
	})
}

// FindSpecialAccount returns the *Account for the admin account with the given
// name. This may be called from any account.
func (c *Config) FindSpecialAccount(ctx context.Context, name string) (*Account, error) {
	return c.FindAccount(ctx, func(a *Account) bool {
		return a.Tags[tagging.Name] == name || a.Tags[tagging.SubstrateSpecialAccount] == name // either tag works; we set both // FIXME maybe just look at SubstrateSpecialAccount now that we're adopting accounts and tolerating Name being potentially overloaded
	})
}

// FindSubstrateAccount returns the *Account for the Substrate account or nil
// (with a nil error) if the organization has not yet run `substrate setup`.
// This may be called from any account.
func (c *Config) FindSubstrateAccount(ctx context.Context) (*Account, error) {
	return c.FindAccount(ctx, func(a *Account) bool {
		return a.Tags[tagging.SubstrateType] == tagging.Substrate
	})
}

// ListAccounts returns all the accounts in the organization in a single slice.
// For a higher-level interface, see accounts.Grouped. This may be called from
// any account.
func (c *Config) ListAccounts(ctx context.Context) (accounts []*Account, err error) {
	if accounts, err = c.listCachedAccounts(); accounts != nil || err != nil {
		return
	}
	ui.Spin("fetching a list of all your AWS accounts and their tags")

	var cfg *Config
	if cfg, err = c.OrganizationReader(ctx); err != nil {
		ui.StopErr(err)
		return
	}
	client := cfg.Organizations()

	// List all the accounts in the organization, even across multiple pages.
	var nextToken *string
	for {
		var out *organizations.ListAccountsOutput
		if out, err = client.ListAccounts(ctx, &organizations.ListAccountsInput{
			NextToken: nextToken,
		}); err != nil {
			ui.StopErr(err)
			return
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
			ui.StopErr(err)
			return
		}
	}

	// Cache the full ListAccounts and ListTagsForResource amalgamation. Do not
	// treat an error here as fatal, since all it would do is slow us down.
	if pathname, err := fileutil.PathnameInParents(AccountsFilename); err == nil {
		pathname = filepath.Join(filepath.Dir(pathname), CachedAccountsFilename)
		if err := jsonutil.Write(accounts, pathname); err != nil {
			ui.Print(err)
		}
	}

	// Memoize the list of accounts and tags, too.
	c.memoizeAccounts(accounts)

	ui.StopErr(err)
	return
}

func (c *Config) listCachedAccounts() ([]*Account, error) {
	if c.accounts != nil && c.accountsExpiry.After(time.Now()) {
		return c.accounts, nil
	}

	if pathname, err := fileutil.PathnameInParents(CachedAccountsFilename); err == nil {
		var accounts []*Account
		if err := jsonutil.Read(pathname, &accounts); err == nil {
			c.memoizeAccounts(accounts)
			return accounts, nil
		}
	}

	return nil, nil // cache miss
}

func (c *Config) memoizeAccounts(accounts []*Account) {
	c.accounts = accounts
	c.accountsExpiry = time.Now().Add(MemoizedAccountsTTL)
}
