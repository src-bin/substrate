package awsorgs

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
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
	organizations.Account
	Tags map[string]string
}

func (a *Account) String() string {
	return jsonutil.MustString(a)
}

type AccountNotFound string

func (err AccountNotFound) Error() string {
	return fmt.Sprintf("account not found: %s", string(err))
}

func DescribeAccount(svc *organizations.Organizations, accountId string) (*Account, error) {
	in := &organizations.DescribeAccountInput{
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
	svc *organizations.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	domain, environment, quality string,
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
	)
}

func EnsureSpecialAccount(
	svc *organizations.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	name string,
) (*Account, error) {
	return ensureAccount(svc, qsvc, name, map[string]string{
		tags.Manager:                 tags.Substrate,
		tags.Name:                    name,
		tags.SubstrateSpecialAccount: name, // TODO get rid of this
		tags.SubstrateVersion:        version.Version,
	})
}

func FindAccount(
	svc *organizations.Organizations,
	domain, environment, quality string,
) (*Account, error) {
	return FindAccountByName(svc, NameFor(domain, environment, quality))
}

func FindAccountsByDomain(
	svc *organizations.Organizations,
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

func FindAccountByName(svc *organizations.Organizations, name string) (*Account, error) {
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

func FindSpecialAccount(svc *organizations.Organizations, name string) (*Account, error) {
	return FindAccountByName(svc, name)
}

func ListAccounts(svc *organizations.Organizations) (accounts []*Account, err error) {
	var nextToken *string
	for {
		in := &organizations.ListAccountsInput{NextToken: nextToken}
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
	svc *organizations.Organizations,
	accountId string,
	tags map[string]string,
) error {
	tagStructs := make([]*organizations.Tag, 0, len(tags))
	for key, value := range tags {
		tagStructs = append(tagStructs, &organizations.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	in := &organizations.TagResourceInput{
		ResourceId: aws.String(accountId),
		Tags:       tagStructs,
	}
	_, err := svc.TagResource(in)
	return err
}

func createAccount(
	svc *organizations.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	name, email string,
) (*organizations.CreateAccountStatus, error) {

	in := &organizations.CreateAccountInput{
		AccountName: aws.String(name),
		Email:       aws.String(email),
	}
	var (
		out *organizations.CreateAccountOutput
		err error
	)
	for {
		out, err = svc.CreateAccount(in)

		// If we're at the organization's limit on the number of AWS accounts
		// it can contain, raise the limit and retry.
		if cveErr, ok := err.(*organizations.ConstraintViolationException); ok && aws.StringValue(cveErr.Reason) == ACCOUNT_NUMBER_LIMIT_EXCEEDED {
			accounts, err := ListAccounts(svc)
			if err != nil {
				return nil, err
			}
			if err := awsservicequotas.EnsureServiceQuota(
				qsvc,
				"L-29A0C5DF", "organizations", // AWS accounts in an organization
				float64(len(accounts)+1),
				float64(len(accounts)*2), // avoid dealing with service limits very often
				time.Time{},
			); err != nil {
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
		in := &organizations.DescribeCreateAccountStatusInput{
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

func emailFor(svc *organizations.Organizations, name string) (string, error) {
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
	svc *organizations.Organizations,
	qsvc *servicequotas.ServiceQuotas,
	name string,
	tags map[string]string,
) (*Account, error) {

	email, err := emailFor(svc, name)
	if err != nil {
		return nil, err
	}

	status, err := createAccount(svc, qsvc, name, email)
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

	return DescribeAccount(svc, accountId)
}

func listTagsForResource(
	svc *organizations.Organizations,
	accountId string,
) (map[string]string, error) {
	var nextToken *string
	tags := make(map[string]string)
	for {
		in := &organizations.ListTagsForResourceInput{
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
