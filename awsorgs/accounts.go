package awsorgs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	organizationsv1 "github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	ConstraintViolationException    = "ConstraintViolationException"
	FinalizingOrganizationException = "FinalizingOrganizationException"
)

type Account = awscfg.Account

type AccountV1 struct {
	organizationsv1.Account
	Tags tags.Tags
}

func (a *AccountV1) String() string {
	return jsonutil.MustString(a)
}

type AccountNotFound string

func (err AccountNotFound) Error() string {
	return fmt.Sprintf("account not found: %s", string(err))
}

type CreateAccountStatus = types.CreateAccountStatus

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

func DescribeAccountV1(svc *organizationsv1.Organizations, accountId string) (*AccountV1, error) {
	in := &organizationsv1.DescribeAccountInput{
		AccountId: aws.String(accountId),
	}
	out, err := svc.DescribeAccount(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	tags, err := listTagsForResourceV1(svc, accountId)
	if err != nil {
		return nil, err
	}
	return &AccountV1{Account: *out.Account, Tags: tags}, nil
}

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
		tags.Tags{
			tags.Domain:           domain,
			tags.Environment:      environment,
			tags.Manager:          tags.Substrate,
			tags.Name:             NameFor(domain, environment, quality),
			tags.Quality:          quality,
			tags.SubstrateVersion: version.Version,
		},
		deadline,
	)
}

func EnsureSpecialAccount(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
) (*Account, error) {
	return ensureAccount(ctx, cfg, name, tags.Tags{
		tags.Manager:                 tags.Substrate,
		tags.Name:                    name,
		tags.SubstrateSpecialAccount: name, // TODO get rid of this
		tags.SubstrateVersion:        version.Version,
	}, time.Time{})
}

func FindAccount(
	svc *organizationsv1.Organizations,
	domain, environment, quality string,
) (*AccountV1, error) {
	return FindAccountByName(svc, NameFor(domain, environment, quality))
}

func FindAccountsByDomain(
	svc *organizationsv1.Organizations,
	domain string,
) (accounts []*AccountV1, err error) {
	allAccounts, err := ListAccountsV1(svc)
	if err != nil {
		return nil, err
	}
	for _, account := range allAccounts {
		if account.Tags[tags.Domain] == domain {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

func FindAccountByName(svc *organizationsv1.Organizations, name string) (*AccountV1, error) {
	accounts, err := ListAccountsV1(svc)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if aws.ToString(account.Name) == name {
			return account, nil
		}
	}
	return nil, AccountNotFound(name)
}

func FindSpecialAccount(svc *organizationsv1.Organizations, name string) (*AccountV1, error) {
	return FindAccountByName(svc, name)
}

func ListAccounts(ctx context.Context, cfg *awscfg.Config) (accounts []*Account, err error) {
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
			tags, err := listTagsForResource(ctx, cfg, aws.ToString(account.Id))
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, &Account{Account: account, Tags: tags})
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func ListAccountsV1(svc *organizationsv1.Organizations) ([]*AccountV1, error) {

	// TODO manage a cache here to buy back some performance if we need it.
	// Might see benefits by caching for even one minute. Invalidation rules
	// are simple: Proactively clear it on account creation. Refresh it on
	// cache miss. Don't worry about staleness since we won't be looking for
	// an account that's been closed and anyway we'll figure it out pretty
	// quickly if we try to access it.

	return ListAccountsV1Fresh(svc)
}

func ListAccountsV1Fresh(svc *organizationsv1.Organizations) (accounts []*AccountV1, err error) {
	var nextToken *string
	for {
		in := &organizationsv1.ListAccountsInput{NextToken: nextToken}
		out, err := svc.ListAccounts(in)
		if err != nil {
			return nil, err
		}
		for _, rawAccount := range out.Accounts {
			tags, err := listTagsForResourceV1(svc, aws.ToString(rawAccount.Id))
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, &AccountV1{Account: *rawAccount, Tags: tags})
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
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

func Tag(
	ctx context.Context,
	cfg *awscfg.Config,
	accountId string,
	tags tags.Tags,
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
			AccountName: aws.String(name),
			Email:       aws.String(email),
		})

		// If we're at the organization's limit on the number of AWS accounts
		// it can contain, raise the limit and retry.
		if cveErr, ok := err.(*types.ConstraintViolationException); ok && cveErr.Reason == types.ConstraintViolationExceptionReasonAccountNumberLimitExceeded {
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

func ensureAccount(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	tagMap tags.Tags, // TODO rename back to tags once the tags package is renamed to tagging
	deadline time.Time,
) (*Account, error) {

	email, err := emailFor(ctx, cfg, name)
	if err != nil {
		return nil, err
	}

	status, err := createAccount(ctx, cfg, name, email, deadline)
	if err != nil {
		return nil, err
	}
	var accountId string
	if status.FailureReason == types.CreateAccountFailureReasonEmailAlreadyExists {
		account, err := cfg.FindAccount(ctx, func(a *awscfg.Account) bool {
			return aws.ToString(a.Name) == name // confirms name matches in addition to email
			// TODO confirm tags match also/instead
		})
		if err != nil {
			return nil, err
		}
		accountId = aws.ToString(account.Id)
	} else {
		accountId = aws.ToString(status.AccountId)
	}

	if err := Tag(ctx, cfg, accountId, tagMap); err != nil {
		return nil, err
	}

	return DescribeAccount(ctx, cfg, accountId)
}

func listTagsForResource(ctx context.Context, cfg *awscfg.Config, accountId string) (tags.Tags, error) {
	client := cfg.Organizations()
	var nextToken *string
	tags := make(tags.Tags)
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

func listTagsForResourceV1(
	svc *organizationsv1.Organizations,
	accountId string,
) (map[string]string, error) {
	var nextToken *string
	tags := make(map[string]string)
	for {
		in := &organizationsv1.ListTagsForResourceInput{
			NextToken:  nextToken,
			ResourceId: aws.String(accountId),
		}
		out, err := svc.ListTagsForResource(in)
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
