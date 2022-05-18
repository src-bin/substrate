package awsorgs

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	organizationsv1 "github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	ACCOUNT_NUMBER_LIMIT_EXCEEDED   = "ACCOUNT_NUMBER_LIMIT_EXCEEDED"
	ConstraintViolationException    = "ConstraintViolationException"
	EMAIL_ALREADY_EXISTS            = "EMAIL_ALREADY_EXISTS"
	FAILED                          = "FAILED"
	FinalizingOrganizationException = "FinalizingOrganizationException"
	SUCCEEDED                       = "SUCCEEDED"
)

type Account struct {
	organizationsv1.Account
	Tags map[string]string
}

func (a *Account) String() string {
	return jsonutil.MustString(a)
}

type AccountNotFound string

func (err AccountNotFound) Error() string {
	return fmt.Sprintf("account not found: %s", string(err))
}

func DescribeAccountV1(svc *organizationsv1.Organizations, accountId string) (*Account, error) {
	in := &organizationsv1.DescribeAccountInput{
		AccountId: aws.String(accountId),
	}
	out, err := svc.DescribeAccount(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	tags, err := listTagsForResource(svc, accountId)
	if err != nil {
		return nil, err
	}
	return &Account{Account: *out.Account, Tags: tags}, nil
}

func EnsureAccount(
	svc *organizationsv1.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	domain, environment, quality string,
	deadline time.Time,
) (*Account, error) {
	return ensureAccount(
		svc,
		qsvc,
		NameFor(domain, environment, quality),
		map[string]string{
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
	svc *organizationsv1.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	name string,
) (*Account, error) {
	return ensureAccount(svc, qsvc, name, map[string]string{
		tags.Manager:                 tags.Substrate,
		tags.Name:                    name,
		tags.SubstrateSpecialAccount: name, // TODO get rid of this
		tags.SubstrateVersion:        version.Version,
	}, time.Time{})
}

func FindAccount(
	svc *organizationsv1.Organizations,
	domain, environment, quality string,
) (*Account, error) {
	return FindAccountByName(svc, NameFor(domain, environment, quality))
}

func FindAccountsByDomain(
	svc *organizationsv1.Organizations,
	domain string,
) (accounts []*Account, err error) {
	allAccounts, err := ListAccounts(svc)
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

func FindAccountByName(svc *organizationsv1.Organizations, name string) (*Account, error) {
	accounts, err := ListAccounts(svc)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if aws.StringValue(account.Name) == name {
			return account, nil
		}
	}
	return nil, AccountNotFound(name)
}

func FindSpecialAccount(svc *organizationsv1.Organizations, name string) (*Account, error) {
	return FindAccountByName(svc, name)
}

func ListAccounts(svc *organizationsv1.Organizations) ([]*Account, error) {

	// TODO manage a cache here to buy back some performance if we need it.
	// Might see benefits by caching for even one minute. Invalidation rules
	// are simple: Proactively clear it on account creation. Refresh it on
	// cache miss. Don't worry about staleness since we won't be looking for
	// an account that's been closed and anyway we'll figure it out pretty
	// quickly if we try to access it.

	return ListAccountsFresh(svc)
}

func ListAccountsFresh(svc *organizationsv1.Organizations) (accounts []*Account, err error) {
	var nextToken *string
	for {
		in := &organizationsv1.ListAccountsInput{NextToken: nextToken}
		out, err := svc.ListAccounts(in)
		if err != nil {
			return nil, err
		}
		for _, rawAccount := range out.Accounts {
			tags, err := listTagsForResource(svc, aws.StringValue(rawAccount.Id))
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, &Account{Account: *rawAccount, Tags: tags})
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
	svc *organizationsv1.Organizations,
	accountId string,
	tags map[string]string,
) error {
	tagStructs := make([]*organizationsv1.Tag, 0, len(tags))
	for key, value := range tags {
		tagStructs = append(tagStructs, &organizationsv1.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	in := &organizationsv1.TagResourceInput{
		ResourceId: aws.String(accountId),
		Tags:       tagStructs,
	}
	_, err := svc.TagResource(in)
	//log.Printf("awsorgs.Tag accountId: %v, in: %+v, out: %+v, err: %v", accountId, in, out, err)
	return err
}

func createAccount(
	svc *organizationsv1.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	name, email string,
	deadline time.Time,
) (*organizationsv1.CreateAccountStatus, error) {

	in := &organizationsv1.CreateAccountInput{
		AccountName: aws.String(name),
		Email:       aws.String(email),
	}
	var (
		out *organizationsv1.CreateAccountOutput
		err error
	)
	for {
		out, err = svc.CreateAccount(in)

		// If we're at the organization's limit on the number of AWS accounts
		// it can contain, raise the limit and retry.
		if cveErr, ok := err.(*organizationsv1.ConstraintViolationException); ok && aws.StringValue(cveErr.Reason) == ACCOUNT_NUMBER_LIMIT_EXCEEDED {
			accounts, err := ListAccounts(svc)
			if err != nil {
				return nil, err
			}
			if err := awsservicequotas.EnsureServiceQuota(
				qsvc,
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
		in := &organizationsv1.DescribeCreateAccountStatusInput{
			CreateAccountRequestId: status.Id,
		}
		out, err := svc.DescribeCreateAccountStatus(in)
		if err != nil {
			return nil, err
		}
		//log.Printf("%+v", out)
		status = out.CreateAccountStatus
		if state := aws.StringValue(out.CreateAccountStatus.State); state == FAILED {
			break // return nil, fmt.Errorf("account creation failed for the %s account with reason %s", name, reason)
		} else if state == SUCCEEDED {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	time.Sleep(10e9) // give it a moment with itself so an AssumeRole immediately after this function returns actually works (TODO do it gracefully)
	return status, nil
}

func emailFor(svc *organizationsv1.Organizations, name string) (string, error) {
	org, err := DescribeOrganization(svc)
	if err != nil {
		return "", err
	}
	return strings.Replace(
		aws.StringValue(org.MasterAccountEmail),
		"@",
		fmt.Sprintf("+%s@", name),
		1,
	), nil
}

func ensureAccount(
	svc *organizationsv1.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	name string,
	tags map[string]string,
	deadline time.Time,
) (*Account, error) {

	email, err := emailFor(svc, name)
	if err != nil {
		return nil, err
	}

	status, err := createAccount(svc, qsvc, name, email, deadline)
	if err != nil {
		return nil, err
	}
	var accountId string
	if reason := aws.StringValue(status.FailureReason); reason == EMAIL_ALREADY_EXISTS {
		account, err := FindAccountByName(svc, name) // confirms name matches in addition to email
		if err != nil {
			return nil, err
		}
		accountId = aws.StringValue(account.Id)
	} else {
		accountId = aws.StringValue(status.AccountId)
	}

	if err := Tag(svc, accountId, tags); err != nil {
		return nil, err
	}

	return DescribeAccountV1(svc, accountId)
}

func listTagsForResource(
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
			tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return tags, nil
}
