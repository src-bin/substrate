package accounts

import (
	"os"
	"text/template"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
)

const (
	Admin   = "admin"
	Audit   = "audit"
	Deploy  = "deploy"
	Master  = "master"
	Network = "network"

	Filename = "substrate.accounts.txt"
)

func CheatSheet(
	org *organizations.Organization,
	auditAccount, deployAccount, networkAccount *awsorgs.Account,
	// TODO adminAccounts []*organizations.Account,
) error {
	f, err := os.Create(Filename)
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("accounts").Parse(cheatSheetTemplate())
	if err != nil {
		return err
	}
	return tmpl.Execute(f, struct {
		Organization                                *organizations.Organization
		AuditAccount, DeployAccount, NetworkAccount *awsorgs.Account
	}{org, auditAccount, deployAccount, networkAccount})
}

func cheatSheetTemplate() string {
	return `Welcome to your Substrate-managed AWS organization!

You can find the Substrate documentation at <https://src-bin.co/substrate.html>.

You're likely to want to use the AWS CLI or Console to explore and manipulate
your Organization.  Here are the account numbers and roles you'll need:

+--------------+----------------+---------------------------+----------------------------------------------------------+
+ Account Name | Account Number | Role Name                 | Role ARN                                                 |
+--------------+----------------+---------------------------+----------------------------------------------------------+
| master       | {{.Organization.MasterAccountId}}   | OrganizationAdministrator | arn:aws:iam::{{.Organization.MasterAccountId}}:role/OrganizationAdministrator |
| audit        | {{.AuditAccount.Id}}   | Auditor (soon)            | arn:aws:iam::{{.AuditAccount.Id}}:role/Auditor (soon)            |
| deploy       | {{.DeployAccount.Id}}   | DeployAdministrator       | arn:aws:iam::{{.DeployAccount.Id}}:role/DeployAdministrator       |
| network      | {{.NetworkAccount.Id}}   | NetworkAdministrator      | arn:aws:iam::{{.NetworkAccount.Id}}:role/NetworkAdministrator      |
+--------------+----------------+---------------------------+----------------------------------------------------------+

There's also at least one admin account where you'll land when you get an
instance from the instance factory or login to the AWS Console.
`
}
