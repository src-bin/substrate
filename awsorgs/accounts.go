package awsorgs

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/version"
)

const (
	EMAIL_ALREADY_EXISTS = "EMAIL_ALREADY_EXISTS"
	FAILED               = "FAILED"
	SUCCEEDED            = "SUCCEEDED"
)

func DescribeAccount(svc *organizations.Organizations, accountId string) (*organizations.Account, error) {
	in := &organizations.DescribeAccountInput{
		AccountId: aws.String(accountId),
	}
	out, err := svc.DescribeAccount(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.Account, nil
}

func EmailForAccount(org *organizations.Organization, accountName string) string {
	return strings.Replace(
		aws.StringValue(org.MasterAccountEmail),
		"@",
		fmt.Sprintf("+%s@", accountName),
		1,
	)
}

func EnsureAccount(
	svc *organizations.Organizations,
	domain, environment, quality, email string,
) (*organizations.Account, error) {
	return ensureAccount(
		svc,
		nameFor(domain, environment, quality),
		email,
		map[string]string{
			tags.Domain:           domain,
			tags.Environment:      environment,
			tags.Manager:          tags.Substrate,
			tags.Quality:          quality,
			tags.SubstrateVersion: version.Version,
		},
	)
}

func EnsureSpecialAccount(
	svc *organizations.Organizations,
	name, email string,
) (*organizations.Account, error) {
	return ensureAccount(svc, name, email, map[string]string{
		tags.Manager:                 tags.Substrate,
		tags.SubstrateSpecialAccount: name,
		tags.SubstrateVersion:        version.Version,
	})
}

func FindAccount(
	svc *organizations.Organizations,
	domain, environment, quality string,
) (*organizations.Account, error) {
	return FindSpecialAccount(svc, nameFor(domain, environment, quality))
}

func FindSpecialAccount(svc *organizations.Organizations, name string) (*organizations.Account, error) {
	return findAccount(svc, name)
}

func RegisterDelegatedAdministrator(svc *organizations.Organizations, accountId, service string) error {
	in := &organizations.RegisterDelegatedAdministratorInput{
		AccountId:        aws.String(accountId),
		ServicePrincipal: aws.String(service),
	}
	_, err := svc.RegisterDelegatedAdministrator(in)
	return err
}

func Tag(svc *organizations.Organizations, accountId string, tags map[string]string) error {
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
	name, email string,
) (*organizations.CreateAccountStatus, error) {

	in := &organizations.CreateAccountInput{
		AccountName: aws.String(name),
		Email:       aws.String(email),
	}
	out, err := svc.CreateAccount(in)
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
	return status, nil
}

func ensureAccount(
	svc *organizations.Organizations,
	name, email string,
	tags map[string]string,
) (*organizations.Account, error) {

	status, err := createAccount(svc, name, email)
	if err != nil {
		return nil, err
	}
	var accountId string
	if reason := aws.StringValue(status.FailureReason); reason == EMAIL_ALREADY_EXISTS {
		account, err := findAccount(svc, name) // confirms name matches in addition to email
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

func findAccount(svc *organizations.Organizations, name string) (*organizations.Account, error) {
	for account := range listAccounts(svc) {
		if aws.StringValue(account.Name) == name {
			return account, nil
		}
	}
	return nil, fmt.Errorf("%s account not found", name)
}

func listAccounts(svc *organizations.Organizations) <-chan *organizations.Account {
	ch := make(chan *organizations.Account)
	go func(chan<- *organizations.Account) {
		var nextToken *string
		for {
			in := &organizations.ListAccountsInput{NextToken: nextToken}
			out, err := svc.ListAccounts(in)
			if err != nil {
				log.Fatal(err)
			}
			for _, account := range out.Accounts {
				ch <- account
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
		close(ch)
	}(ch)
	return ch
}

func nameFor(domain, environment, quality string) string {
	return fmt.Sprintf("%s-%s-%s", domain, environment, quality)
}
