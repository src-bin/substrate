package accounts

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
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

	adminAccountsCells := ui.MakeTableCells(5, 1)
	adminAccountsCells[0][0] = "Quality"
	adminAccountsCells[0][1] = "Account Number"
	adminAccountsCells[0][2] = "Role Name"
	adminAccountsCells[0][3] = "Role ARN"
	adminAccountsCells[0][4] = "E-mail"
	serviceAccountsCells := ui.MakeTableCells(7, 1)
	serviceAccountsCells[0][0] = "Domain"
	serviceAccountsCells[0][1] = "Environment"
	serviceAccountsCells[0][2] = "Quality"
	serviceAccountsCells[0][3] = "Account Number"
	serviceAccountsCells[0][4] = "Role Name"
	serviceAccountsCells[0][5] = "Role ARN"
	serviceAccountsCells[0][6] = "E-mail"
	specialAccountsCells := ui.MakeTableCells(5, 6)
	specialAccountsCells[0][0] = "Account Name"
	specialAccountsCells[0][1] = "Account Number"
	specialAccountsCells[0][2] = "Role Name"
	specialAccountsCells[0][3] = "Role ARN"
	specialAccountsCells[0][4] = "E-mail"

	adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := Grouped(svc)
	if err != nil {
		return err
	}

	specialAccountsCells[3][0] = Audit
	specialAccountsCells[3][1] = aws.StringValue(auditAccount.Id)
	specialAccountsCells[3][2] = roles.Auditor
	specialAccountsCells[3][3] = roles.Arn(aws.StringValue(auditAccount.Id), roles.Auditor)
	specialAccountsCells[3][4] = aws.StringValue(auditAccount.Email)

	specialAccountsCells[4][0] = Deploy
	specialAccountsCells[4][1] = aws.StringValue(deployAccount.Id)
	specialAccountsCells[4][2] = roles.DeployAdministrator
	specialAccountsCells[4][3] = roles.Arn(aws.StringValue(deployAccount.Id), roles.DeployAdministrator)
	specialAccountsCells[4][4] = aws.StringValue(deployAccount.Email)

	specialAccountsCells[1][0] = Management
	specialAccountsCells[1][1] = aws.StringValue(managementAccount.Id)
	specialAccountsCells[1][2] = roles.OrganizationAdministrator
	specialAccountsCells[1][3] = roles.Arn(aws.StringValue(managementAccount.Id), roles.OrganizationAdministrator)
	specialAccountsCells[1][4] = aws.StringValue(managementAccount.Email)

	specialAccountsCells[5][0] = Network
	specialAccountsCells[5][1] = aws.StringValue(networkAccount.Id)
	specialAccountsCells[5][2] = roles.NetworkAdministrator
	specialAccountsCells[5][3] = roles.Arn(aws.StringValue(networkAccount.Id), roles.NetworkAdministrator)
	specialAccountsCells[5][4] = aws.StringValue(networkAccount.Email)

	for _, account := range adminAccounts {
		adminAccountsCells = append(adminAccountsCells, []string{
			account.Tags[tags.Quality],
			aws.StringValue(account.Id),
			roles.Administrator,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
			aws.StringValue(account.Email),
		})
	}

	for _, account := range serviceAccounts {
		serviceAccountsCells = append(serviceAccountsCells, []string{
			account.Tags[tags.Domain],
			account.Tags[tags.Environment],
			account.Tags[tags.Quality],
			aws.StringValue(account.Id),
			roles.Administrator,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
			aws.StringValue(account.Email),
		})
	}

	fmt.Fprint(f, "Welcome to your Substrate-managed AWS organization!\n")
	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "You can find the Substrate documentation at <https://src-bin.com/substrate/>.\n")
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

	// Try to get the authoritative order of environments and qualities from
	// substrate.{environments,qualities}. We won't have access to that in
	// Lambda, though, so we've got to come up with something. I decided to
	// guess based on how I see folks using Substrate, how I advise folks to
	// use Substrate, and how folks name environments and release channels in
	// the broader world. I hope Substrate thrives sufficiently for me to
	// regret this.
	var environments, qualities []string
	if environments, err = naming.Environments(); err != nil {
		environments = []string{
			"dev", "devel", "development",
			"qa", "test", "testing",
			"stage", "staging",
			"prod", "production",
		}
	}
	if qualities, err = naming.Qualities(); err != nil {
		qualities = []string{
			"alpha",
			"beta", "canary",
			"gamma", "default",
		}
	}
	err = nil

	sort.Slice(adminAccounts, func(i, j int) bool {
		return searchUnsorted(qualities, adminAccounts[i].Tags[tags.Quality]) < searchUnsorted(qualities, adminAccounts[j].Tags[tags.Quality])
	})
	sort.Slice(serviceAccounts, func(i, j int) bool {
		if serviceAccounts[i].Tags[tags.Domain] != serviceAccounts[j].Tags[tags.Domain] {
			return serviceAccounts[i].Tags[tags.Domain] < serviceAccounts[j].Tags[tags.Domain]
		}
		if serviceAccounts[i].Tags[tags.Environment] != serviceAccounts[j].Tags[tags.Environment] {
			return searchUnsorted(environments, serviceAccounts[i].Tags[tags.Environment]) < searchUnsorted(environments, serviceAccounts[j].Tags[tags.Environment])
		}
		if serviceAccounts[i].Tags[tags.Quality] != serviceAccounts[j].Tags[tags.Quality] {
			return searchUnsorted(qualities, serviceAccounts[i].Tags[tags.Quality]) < searchUnsorted(qualities, serviceAccounts[j].Tags[tags.Quality])
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
	return nil
}

// searchUnsorted is like sort.SearchStrings but it allows for the search
// space to be unsorted and assumes it's pretty small.
func searchUnsorted(a []string, x string) int {
	for i, s := range a {
		if s == x {
			return i
		}
	}
	return -1
}
