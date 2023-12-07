package awsorgs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	AccountNotFoundException        = "AccountNotFoundException"
	ConstraintViolationException    = "ConstraintViolationException"
	FinalizingOrganizationException = "FinalizingOrganizationException"
)

type Account = awscfg.Account

// StringableZeroAccount returns an *Account with just enough fields filled in
// to allow it to respond to the String method.
func StringableZeroAccount(number string) *Account {
	return &Account{
		Account: types.Account{Id: aws.String(number)},
		Tags:    tagging.Map{},
	}
}

type AccountNotFound string

func (err AccountNotFound) Error() string {
	return fmt.Sprintf("account not found: %s", string(err))
}

type CreateAccountStatus = types.CreateAccountStatus

// DescribeAccount fetches an account from the AWS Organizations API and
// augments it with its tags. The *Config must be in the management account.
func DescribeAccount(ctx context.Context, cfg *awscfg.Config, accountId string) (*Account, error) {
	out, err := cfg.Organizations().DescribeAccount(ctx, &organizations.DescribeAccountInput{
		AccountId: aws.String(accountId),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	tags, err := listTagsForResource(ctx, cfg, accountId)
	if err != nil {
		return nil, err
	}
	return &Account{Account: *out.Account, Tags: tags}, nil
}

// EnsureAccount creates an AWS account tagged with a domain, environment, and
// quality. The *Config must be in the management account.
func EnsureAccount(
	ctx context.Context,
	cfg *awscfg.Config,
	domain, environment, quality string,
	deadline time.Time,
) (*Account, error) {
	return ensureAccount(
		ctx,
		cfg,
		NameFor(domain, environment, quality),
		tagging.Map{
			tagging.Domain:           domain,
			tagging.Environment:      environment,
			tagging.Manager:          tagging.Substrate,
			tagging.Name:             NameFor(domain, environment, quality),
			tagging.Quality:          quality,
			tagging.SubstrateVersion: version.Version,
		},
		deadline,
	)
}

// EnsureSpecialAccount creates a named AWS account. The *Config must be in the
// management account.
func EnsureSpecialAccount(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
) (*Account, error) {
	tags := tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.Name:             name,
		tagging.SubstrateType:    name, // this may not be completely correct but works for "Substrate" and "audit", at least
		tagging.SubstrateVersion: version.Version,
	}
	if name != tagging.Substrate {
		tags[tagging.SubstrateSpecialAccount] = name
	}
	return ensureAccount(ctx, cfg, name, tags, time.Time{})
}

func ListAccounts(ctx context.Context, cfg *awscfg.Config) ([]*Account, error) {
	return cfg.ListAccounts(ctx)
}

func Must(account *Account, err error) *Account {
	if err != nil {
		ui.Fatal(err)
	}
	return account
}

func NameFor(domain, environment, quality string) string {
	if domain == environment {
		return fmt.Sprintf("%s-%s", environment, quality)
	}
	return fmt.Sprintf("%s-%s-%s", domain, environment, quality)
}

// Tag tags an AWS account. The *Config must be in the management account.
func Tag(
	ctx context.Context,
	cfg *awscfg.Config,
	accountId string,
	tags tagging.Map,
) error {
	tagStructs := make([]types.Tag, 0, len(tags))
	for key, value := range tags {
		tagStructs = append(tagStructs, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	_, err := cfg.Organizations().TagResource(ctx, &organizations.TagResourceInput{
		ResourceId: aws.String(accountId),
		Tags:       tagStructs,
	})
	return err
}

func createAccount(
	ctx context.Context,
	cfg *awscfg.Config,
	name, email string,
	deadline time.Time,
) (*CreateAccountStatus, error) {
	client := cfg.Organizations()
	var (
		out *organizations.CreateAccountOutput
		err error
	)
	for {
		out, err = client.CreateAccount(ctx, &organizations.CreateAccountInput{
			AccountName:            aws.String(name),
			Email:                  aws.String(email),
			IamUserAccessToBilling: types.IAMUserAccessToBillingAllow,
		})

		// If we're at the organization's limit on the number of AWS accounts
		// it can contain, raise the limit and retry.
		var cveErr *types.ConstraintViolationException
		if errors.As(err, &cveErr) && cveErr.Reason == types.ConstraintViolationExceptionReasonAccountNumberLimitExceeded {
			accounts, err := ListAccounts(ctx, cfg)
			if err != nil {
				return nil, err
			}
			if err := awsservicequotas.EnsureServiceQuota(
				ctx,
				cfg.Regional("us-east-1"),     // quotas for global services must be managed in us-east-1
				"L-29A0C5DF", "organizations", // AWS accounts in an organization
				float64(len(accounts)+1),
				float64(len(accounts)*2), // avoid dealing with service limits very often
				deadline,
			); err != nil {
				if _, ok := err.(awsservicequotas.DeadlinePassed); ok {
					ui.Print(err)
					break
				}
				return nil, err
			}
			continue
		}

		if !awsutil.ErrorCodeIs(err, FinalizingOrganizationException) {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)

	status := out.CreateAccountStatus
	for {
		out, err := client.DescribeCreateAccountStatus(ctx, &organizations.DescribeCreateAccountStatusInput{
			CreateAccountRequestId: status.Id,
		})
		if err != nil {
			return nil, err
		}
		//log.Printf("%+v", out)
		status = out.CreateAccountStatus
		if status.State == types.CreateAccountStateFailed {
			break // return nil, fmt.Errorf("account creation failed for the %s account with reason %s", name, reason)
		} else if status.State == types.CreateAccountStateSucceeded {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	time.Sleep(10e9) // give it a moment with itself so an AssumeRole immediately after this function returns actually works (TODO do it gracefully)

	if err := cfg.ClearCachedAccounts(); err != nil {
		return nil, err
	}

	return status, nil
}

func emailFor(ctx context.Context, cfg *awscfg.Config, name string) (string, error) {
	org, err := cfg.DescribeOrganization(ctx)
	if err != nil {
		return "", err
	}
	return strings.Replace(
		aws.ToString(org.MasterAccountEmail),
		"@",
		fmt.Sprintf("+%s@", name),
		1,
	), nil
}

// ensureAccount uses FindAccount and createAccount to ensure an account that
// meets the given criteria exists and is tagged appropriately. The given tags
// may be modified by this function.
func ensureAccount(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	tags tagging.Map,
	deadline time.Time,
) (*Account, error) {

	email, err := emailFor(ctx, cfg, name)
	if err != nil {
		return nil, err
	}

	// findAccount is a closure to be passed to cfg.FindAccount during the
	// idempotency guarantees when creating accounts.
	findAccount := func(a *awscfg.Account) bool {
		if tagsEqual(a.Tags, tags) {
			return true
		}
		return aws.ToString(a.Email) == email && aws.ToString(a.Name) == name
	}

	// Try to find the account here, first, even though that looks like a
	// TOCTTOU vulnerability. It isn't, really, since AWS is still enforcing
	// its uniqueness constraint on email address and possibly other aspects
	// of the account. Most importantly, checking this now allows us to return
	// early, without attempting to create the account, which avoids an
	// unnecessary failure when the account exists but we're precisely at
	// the organization's limit on the number of accounts in it.
	account, err := cfg.FindAccount(ctx, findAccount)
	if err != nil {
		return nil, err
	}

	// If we can't find it, try to create it.
	var (
		accountId string
		nameTag   bool // whether to set the Name tag, which we only do on account creation
	)
	if account == nil {
		status, err := createAccount(ctx, cfg, name, email, deadline)
		if err != nil {
			return nil, err
		}
		if status.FailureReason == types.CreateAccountFailureReasonEmailAlreadyExists {
			ui.Debug(status)
			if account, err = cfg.FindAccount(ctx, findAccount); err != nil {
				return nil, err
			}
			if account == nil {
				return nil, AccountNotFound(fmt.Sprintf("<%s> is probably taken by a closed account", email))
			}
			accountId = aws.ToString(account.Id) // found after createAccount failure
		} else {
			accountId = aws.ToString(status.AccountId) // newly-created
			nameTag = true
		}
	} else {
		accountId = aws.ToString(account.Id) // found right away (before even trying to create it)
	}

	// Ensure AWS Config is enabled in this account for every region that's in use.
	// TODO regions.Select()
	// TODO PutConfigurationRecorder
	// TODO PutDeliveryChannel
	// TODO StartConfigurationRecorder

	if !nameTag {
		delete(tags, tagging.Name)
	}
	if err := Tag(ctx, cfg, accountId, tags); err != nil {
		return nil, err
	}

	return DescribeAccount(ctx, cfg, accountId)
}

func listTagsForResource(ctx context.Context, cfg *awscfg.Config, accountId string) (tagging.Map, error) {
	client := cfg.Organizations()
	var nextToken *string
	tags := make(tagging.Map)
	for {
		out, err := client.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
			NextToken:  nextToken,
			ResourceId: aws.String(accountId),
		})
		if err != nil {
			return nil, err
		}
		for _, tag := range out.Tags {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return tags, nil
}

func tagsEqual(a, b tagging.Map) bool {
	if a[tagging.SubstrateSpecialAccount] != "" && a[tagging.SubstrateSpecialAccount] == b[tagging.SubstrateSpecialAccount] {
		return true
	}

	if a[tagging.Domain] == "" || b[tagging.Domain] == "" {
		return false
	}
	if a[tagging.Environment] == "" || b[tagging.Environment] == "" {
		return false
	}
	if a[tagging.Quality] == "" || b[tagging.Quality] == "" {
		return false
	}
	return a[tagging.Domain] == b[tagging.Domain] && a[tagging.Environment] == b[tagging.Environment] && a[tagging.Quality] == b[tagging.Quality]
}
