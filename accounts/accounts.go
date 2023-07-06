package accounts

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/table"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

const (
	Admin      = naming.Admin
	Audit      = naming.Audit
	Deploy     = naming.Deploy
	Management = naming.Management
	Network    = naming.Network

	CheatSheetFilename = awscfg.AccountsFilename
)

func CheatSheet(ctx context.Context, cfg *awscfg.Config) error {

	// TODO If we run `substrate accounts` in any directory except exactly the
	// Substrate repository, this litters extra substrate.accounts.txt files
	// all over the place. We should either stop writing this file entirely
	// (which might require making `substrate accounts` a lot faster) or use
	// PathnameInParents to ensure we write to the correct directory.
	f, err := os.Create(CheatSheetFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	adminAccountsCells := table.MakeCells(6, 1)
	adminAccountsCells[0][0] = "Quality"
	adminAccountsCells[0][1] = "Account Number"
	adminAccountsCells[0][2] = "Role Name"
	adminAccountsCells[0][3] = "Role ARN"
	adminAccountsCells[0][4] = "E-mail"
	adminAccountsCells[0][5] = "Version"
	serviceAccountsCells := table.MakeCells(8, 1)
	serviceAccountsCells[0][0] = "Domain"
	serviceAccountsCells[0][1] = "Environment"
	serviceAccountsCells[0][2] = "Quality"
	serviceAccountsCells[0][3] = "Account Number"
	serviceAccountsCells[0][4] = "Role Name"
	serviceAccountsCells[0][5] = "Role ARN"
	serviceAccountsCells[0][6] = "E-mail"
	serviceAccountsCells[0][7] = "Version"
	specialAccountsCells := table.MakeCells(6, 6)
	specialAccountsCells[0][0] = "Account Name"
	specialAccountsCells[0][1] = "Account Number"
	specialAccountsCells[0][2] = "Role Name"
	specialAccountsCells[0][3] = "Role ARN"
	specialAccountsCells[0][4] = "E-mail"
	specialAccountsCells[0][5] = "Version"

	ui.Must(cfg.ClearCachedAccounts())
	adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := Grouped(ctx, cfg)
	if err != nil {
		return err
	}

	specialAccountsCells[3][0] = Audit
	specialAccountsCells[3][1] = aws.ToString(auditAccount.Id)
	specialAccountsCells[3][2] = roles.Auditor
	specialAccountsCells[3][3] = roles.ARN(aws.ToString(auditAccount.Id), roles.Auditor)
	specialAccountsCells[3][4] = aws.ToString(auditAccount.Email)
	specialAccountsCells[3][5] = auditAccount.Tags[tagging.SubstrateVersion]

	specialAccountsCells[4][0] = Deploy
	specialAccountsCells[4][1] = aws.ToString(deployAccount.Id)
	specialAccountsCells[4][2] = roles.DeployAdministrator
	specialAccountsCells[4][3] = roles.ARN(aws.ToString(deployAccount.Id), roles.DeployAdministrator)
	specialAccountsCells[4][4] = aws.ToString(deployAccount.Email)
	specialAccountsCells[4][5] = deployAccount.Tags[tagging.SubstrateVersion]

	specialAccountsCells[1][0] = Management
	specialAccountsCells[1][1] = aws.ToString(managementAccount.Id)
	specialAccountsCells[1][2] = roles.OrganizationAdministrator
	specialAccountsCells[1][3] = roles.ARN(aws.ToString(managementAccount.Id), roles.OrganizationAdministrator)
	specialAccountsCells[1][4] = aws.ToString(managementAccount.Email)
	specialAccountsCells[1][5] = managementAccount.Tags[tagging.SubstrateVersion]

	specialAccountsCells[5][0] = Network
	specialAccountsCells[5][1] = aws.ToString(networkAccount.Id)
	specialAccountsCells[5][2] = roles.NetworkAdministrator
	specialAccountsCells[5][3] = roles.ARN(aws.ToString(networkAccount.Id), roles.NetworkAdministrator)
	specialAccountsCells[5][4] = aws.ToString(networkAccount.Email)
	specialAccountsCells[5][5] = networkAccount.Tags[tagging.SubstrateVersion]

	for _, account := range adminAccounts {
		adminAccountsCells = append(adminAccountsCells, []string{
			account.Tags[tagging.Quality],
			aws.ToString(account.Id),
			roles.Administrator,
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
			aws.ToString(account.Email),
			account.Tags[tagging.SubstrateVersion],
		})
	}

	for _, account := range serviceAccounts {
		serviceAccountsCells = append(serviceAccountsCells, []string{
			account.Tags[tagging.Domain],
			account.Tags[tagging.Environment],
			account.Tags[tagging.Quality],
			aws.ToString(account.Id),
			roles.Administrator,
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
			aws.ToString(account.Email),
			account.Tags[tagging.SubstrateVersion],
		})
	}

	fmt.Fprint(f, "Welcome to your Substrate-managed AWS organization!\n")
	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "You can find the Substrate documentation at <https://docs.src-bin.com/substrate/>.\n")
	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "You're likely to want to use the AWS CLI or Console to explore and manipulate\n")
	fmt.Fprint(f, "your Organization.  Here are the account numbers and roles you'll need for the\n")
	fmt.Fprint(f, "special accounts that Substrate manages:\n")
	fmt.Fprint(f, "\n")
	table.Ftable(f, specialAccountsCells)

	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "And here are the account numbers and roles for your service accounts:\n")
	fmt.Fprint(f, "\n")
	table.Ftable(f, serviceAccountsCells)

	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "Finally, here are the account numbers and roles for your admin accounts:\n")
	fmt.Fprint(f, "\n")
	table.Ftable(f, adminAccountsCells)

	return nil
}

func Grouped(ctx context.Context, cfg *awscfg.Config) (
	adminAccounts, serviceAccounts []*awsorgs.Account,
	auditAccount, deployAccount, managementAccount, networkAccount *awsorgs.Account,
	err error,
) {
	var allAccounts []*awsorgs.Account
	allAccounts, err = awsorgs.ListAccounts(ctx, cfg)
	if err != nil {
		return
	}

	for _, account := range allAccounts {
		if account.Tags[tagging.SubstrateSpecialAccount] != "" {
			switch account.Tags[tagging.SubstrateSpecialAccount] {
			case Audit:
				auditAccount = account
			case Deploy:
				deployAccount = account
			case Management:
				managementAccount = account
			case Network:
				networkAccount = account
			}
		} else if account.Tags[tagging.Domain] == Admin {
			adminAccounts = append(adminAccounts, account)
		} else {
			serviceAccounts = append(serviceAccounts, account)
		}
	}

	Sort(adminAccounts)
	Sort(serviceAccounts)

	return
}

func Sort(slice []*awsorgs.Account) {

	// Try to get the authoritative order of environments and qualities from
	// substrate.{environments,qualities}. We won't have access to that in
	// Lambda, though, so we've got to come up with something. I decided to
	// guess based on how I see folks using Substrate, how I advise folks to
	// use Substrate, and how folks name environments and release channels in
	// the broader world. I hope Substrate thrives sufficiently for me to
	// regret this.
	var (
		environments, qualities []string
		err                     error
	)
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

	sort.Slice(slice, func(i, j int) bool {
		if slice[i].Tags[tagging.Domain] != slice[j].Tags[tagging.Domain] {
			return slice[i].Tags[tagging.Domain] < slice[j].Tags[tagging.Domain]
		}
		if slice[i].Tags[tagging.Environment] != slice[j].Tags[tagging.Environment] {
			return naming.Index(environments, slice[i].Tags[tagging.Environment]) < naming.Index(environments, slice[j].Tags[tagging.Environment])
		}
		if slice[i].Tags[tagging.Quality] != slice[j].Tags[tagging.Quality] {
			return naming.Index(qualities, slice[i].Tags[tagging.Quality]) < naming.Index(qualities, slice[j].Tags[tagging.Quality])
		}
		if slice[i].Tags[tagging.SubstrateSpecialAccount] != slice[j].Tags[tagging.SubstrateSpecialAccount] {
			return slice[i].Tags[tagging.SubstrateSpecialAccount] < slice[j].Tags[tagging.SubstrateSpecialAccount]
		}
		return false
	})
}
