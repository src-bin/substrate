package accounts

import (
	"fmt"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
)

const (
	Admin   = "admin"
	Audit   = "audit"
	Deploy  = "deploy"
	Master  = "master"
	Network = "network"

	Filename = "substrate.accounts.txt"
)

func CheatSheet(svc *organizations.Organizations) error {
	f, err := os.Create(Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	adminAccountsCells := ui.MakeTableCells(4, 1)
	adminAccountsCells[0][0] = "Quality"
	adminAccountsCells[0][1] = "Account Number"
	adminAccountsCells[0][2] = "Role Name"
	adminAccountsCells[0][3] = "Role ARN"
	otherAccountsCells := ui.MakeTableCells(6, 1)
	otherAccountsCells[0][0] = "Domain"
	otherAccountsCells[0][1] = "Environment"
	otherAccountsCells[0][2] = "Quality"
	otherAccountsCells[0][3] = "Account Number"
	otherAccountsCells[0][4] = "Role Name"
	otherAccountsCells[0][5] = "Role ARN"
	specialAccountsCells := ui.MakeTableCells(4, 6)
	specialAccountsCells[0][0] = "Account Name"
	specialAccountsCells[0][1] = "Account Number"
	specialAccountsCells[0][2] = "Role Name"
	specialAccountsCells[0][3] = "Role ARN"

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
			case Master:
				specialAccountsCells[1][0] = Master
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
			otherAccountsCells = append(otherAccountsCells, []string{
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
	sort.Slice(otherAccountsCells[1:], func(i, j int) bool {
		for k := 0; k <= 2; k++ {
			if otherAccountsCells[i+1][k] != otherAccountsCells[j+1][k] {
				return otherAccountsCells[i+1][k] < otherAccountsCells[j+1][k]
			}
		}
		return false
	})

	fmt.Fprint(f, "Welcome to your Substrate-managed AWS organization!\n")
	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "You can find the Substrate documentation at <https://src-bin.co/substrate.html>.\n")
	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "You're likely to want to use the AWS CLI or Console to explore and manipulate\n")
	fmt.Fprint(f, "your Organization.  Here are the account numbers and roles you'll need for the\n")
	fmt.Fprint(f, "special accounts that Substrate manages:\n")
	fmt.Fprint(f, "\n")
	ui.Ftable(f, specialAccountsCells)

	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "And here are the account numbers and roles for your other accounts:\n")
	fmt.Fprint(f, "\n")
	ui.Ftable(f, otherAccountsCells)

	fmt.Fprint(f, "\n")
	fmt.Fprint(f, "Finally, here are the account numbers and roles for your admin accounts:\n")
	fmt.Fprint(f, "\n")
	ui.Ftable(f, adminAccountsCells)

	return nil
}
