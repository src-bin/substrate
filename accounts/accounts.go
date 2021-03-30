package accounts

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
)

const (
	Admin      = "admin"
	Audit      = "audit"
	Deploy     = "deploy"
	Management = "management"
	Network    = "network"

	CheatSheetFilename             = "substrate.accounts.txt"
	ManagementAccountIdFilename    = "substrate.management-account-id"
	OldManagementAccountIdFilename = "substrate.master-account-id"
)

func CheatSheet(svc *organizations.Organizations) error {
	f, err := os.Create(CheatSheetFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	adminAccountsCells := ui.MakeTableCells(4, 1)
	adminAccountsCells[0][0] = "Quality"
	adminAccountsCells[0][1] = "Account Number"
	adminAccountsCells[0][2] = "Role Name"
	adminAccountsCells[0][3] = "Role ARN"
	serviceAccountsCells := ui.MakeTableCells(6, 1)
	serviceAccountsCells[0][0] = "Domain"
	serviceAccountsCells[0][1] = "Environment"
	serviceAccountsCells[0][2] = "Quality"
	serviceAccountsCells[0][3] = "Account Number"
	serviceAccountsCells[0][4] = "Role Name"
	serviceAccountsCells[0][5] = "Role ARN"
	specialAccountsCells := ui.MakeTableCells(4, 6)
	specialAccountsCells[0][0] = "Account Name"
	specialAccountsCells[0][1] = "Account Number"
	specialAccountsCells[0][2] = "Role Name"
	specialAccountsCells[0][3] = "Role ARN"

	// TODO reimplement this section in terms of the new Grouped function below.
	allAccounts, err := awsorgs.ListAccounts(svc)
	if err != nil {
		return err
	}
	for _, account := range allAccounts {
		if account.Tags[tags.SubstrateSpecialAccount] != "" {
			switch account.Tags[tags.SubstrateSpecialAccount] {
			case Audit:
				specialAccountsCells[3][0] = Audit
				specialAccountsCells[3][1] = aws.StringValue(account.Id)
				specialAccountsCells[3][2] = roles.Auditor
				specialAccountsCells[3][3] = roles.Arn(aws.StringValue(account.Id), roles.Auditor)
			case Deploy:
				specialAccountsCells[4][0] = Deploy
				specialAccountsCells[4][1] = aws.StringValue(account.Id)
				specialAccountsCells[4][2] = roles.DeployAdministrator
				specialAccountsCells[4][3] = roles.Arn(aws.StringValue(account.Id), roles.DeployAdministrator)
			case Management:
				specialAccountsCells[1][0] = Management
				specialAccountsCells[1][1] = aws.StringValue(account.Id)
				specialAccountsCells[1][2] = roles.OrganizationAdministrator
				specialAccountsCells[1][3] = roles.Arn(aws.StringValue(account.Id), roles.OrganizationAdministrator)
			case Network:
				specialAccountsCells[5][0] = Network
				specialAccountsCells[5][1] = aws.StringValue(account.Id)
				specialAccountsCells[5][2] = roles.NetworkAdministrator
				specialAccountsCells[5][3] = roles.Arn(aws.StringValue(account.Id), roles.NetworkAdministrator)
			}
		} else if account.Tags[tags.Domain] == Admin {
			adminAccountsCells = append(adminAccountsCells, []string{
				account.Tags[tags.Quality],
				aws.StringValue(account.Id),
				roles.Administrator,
				roles.Arn(aws.StringValue(account.Id), roles.Administrator),
			})
		} else {
			serviceAccountsCells = append(serviceAccountsCells, []string{
				account.Tags[tags.Domain],
				account.Tags[tags.Environment],
				account.Tags[tags.Quality],
				aws.StringValue(account.Id),
				roles.Administrator,
				roles.Arn(aws.StringValue(account.Id), roles.Administrator),
			})
		}
	}
	sort.Slice(adminAccountsCells[1:], func(i, j int) bool {
		return adminAccountsCells[i+1][0] < adminAccountsCells[j+1][0]
	})
	sort.Slice(serviceAccountsCells[1:], func(i, j int) bool {
		for k := 0; k <= 2; k++ {
			if serviceAccountsCells[i+1][k] != serviceAccountsCells[j+1][k] {
				return serviceAccountsCells[i+1][k] < serviceAccountsCells[j+1][k]
			}
		}
		return false
	})

	fmt.Fprint(f, "Welcome to your Substrate-managed AWS organization!\n")
	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "You can find the Substrate documentation at <https://src-bin.co/substrate/>.\n")
	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "You're likely to want to use the AWS CLI or Console to explore and manipulate\n")
	fmt.Fprint(f, "your Organization.  Here are the account numbers and roles you'll need for the\n")
	fmt.Fprint(f, "special accounts that Substrate manages:\n")
	fmt.Fprint(f, "\n")
	ui.Ftable(f, specialAccountsCells)

	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "And here are the account numbers and roles for your service accounts:\n")
	fmt.Fprint(f, "\n")
	ui.Ftable(f, serviceAccountsCells)

	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "Finally, here are the account numbers and roles for your admin accounts:\n")
	fmt.Fprint(f, "\n")
	ui.Ftable(f, adminAccountsCells)

	return nil
}

func EnsureManagementAccountIdMatchesDisk(managementAccountId string) error {

	// We'll never have this file when we're e.g. in Lambda.
	pathname, err := fileutil.PathnameInParents(ManagementAccountIdFilename)
	if err != nil {
		return nil
	}

	b, err := fileutil.ReadFile(pathname)
	if err != nil {
		return err
	}
	if diskManagementAccountId := fileutil.Tidy(b); managementAccountId != diskManagementAccountId {
		return ManagementAccountMismatchError(fmt.Sprintf(
			"the calling account's management account is %s but this directory's management account is %s",
			managementAccountId,
			diskManagementAccountId,
		))
	}
	return nil
}

func Grouped(svc *organizations.Organizations) (adminAccounts, serviceAccounts []*awsorgs.Account, auditAccount, deployAccount, managementAccount, networkAccount *awsorgs.Account, err error) {
	var allAccounts []*awsorgs.Account
	allAccounts, err = awsorgs.ListAccounts(svc)
	if err != nil {
		return
	}
	for _, account := range allAccounts {
		if account.Tags[tags.SubstrateSpecialAccount] != "" {
			switch account.Tags[tags.SubstrateSpecialAccount] {
			case Audit:
				auditAccount = account
			case Deploy:
				deployAccount = account
			case Management:
				managementAccount = account
			case Network:
				networkAccount = account
			}
		} else if account.Tags[tags.Domain] == Admin {
			adminAccounts = append(adminAccounts, account)
		} else {
			serviceAccounts = append(serviceAccounts, account)
		}
	}
	sort.Slice(adminAccounts, func(i, j int) bool {
		return adminAccounts[i].Tags[tags.Quality] < adminAccounts[j].Tags[tags.Quality]
	})
	sort.Slice(serviceAccounts, func(i, j int) bool {
		if serviceAccounts[i].Tags[tags.Domain] != serviceAccounts[j].Tags[tags.Domain] {
			return serviceAccounts[i].Tags[tags.Domain] < serviceAccounts[j].Tags[tags.Domain]
		}
		if serviceAccounts[i].Tags[tags.Environment] != serviceAccounts[j].Tags[tags.Environment] {
			return serviceAccounts[i].Tags[tags.Environment] < serviceAccounts[j].Tags[tags.Environment]
		}
		if serviceAccounts[i].Tags[tags.Quality] != serviceAccounts[j].Tags[tags.Quality] {
			return serviceAccounts[i].Tags[tags.Quality] < serviceAccounts[j].Tags[tags.Quality]
		}
		return false
	})
	return
}

type ManagementAccountMismatchError string

func (err ManagementAccountMismatchError) Error() string {
	return string(err)
}

func WriteManagementAccountIdToDisk(managementAccountId string) error {
	if !fileutil.Exists(ManagementAccountIdFilename) {
		if err := ioutil.WriteFile(ManagementAccountIdFilename, []byte(fmt.Sprintln(managementAccountId)), 0666); err != nil {
			return err
		}
	}

	// This file used to be stored under a different name. AWS have recently
	// started referring to the account under which the organization was
	// created as the "management" account rather than the "master" account
	// so we reflect that change here.
	//
	// TODO Remove on or after release 2021.01.
	if fileutil.Exists(OldManagementAccountIdFilename) {
		if err := os.Remove(OldManagementAccountIdFilename); err != nil {
			log.Print(err)
		}
	}

	return nil
}
