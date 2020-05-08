package accounts

import (
	"os"
	"text/template"

	"github.com/aws/aws-sdk-go/service/organizations"
)

const (
	Audit   = "audit"
	Deploy  = "deploy"
	Master  = "master"
	Network = "network"
	Ops     = "ops"

	Filename = "substrate.accounts.txt"
)

func CheatSheet(
	org *organizations.Organization,
	auditAccount, deployAccount, networkAccount, opsAccount *organizations.Account,
) error {
	f, err := os.Create(Filename)
	if err != nil {
		return err
	}
	tmpl, err := template.New("accounts").Parse(Template())
	if err != nil {
		return err
	}
	if err = tmpl.Execute(f, struct {
		Organization                                            *organizations.Organization
		AuditAccount, DeployAccount, NetworkAccount, OpsAccount *organizations.Account
	}{org, auditAccount, deployAccount, networkAccount, opsAccount}); err != nil {
		return err
	}
	return nil
}

func Template() string {
	return `Welcome to your Substrate-managed AWS organization!

You can find the Substrate documentation at <https://src-bin.co/substrate.html>.

You're likely to want to use the AWS CLI or Console to explore and manipulate
your Organization.  Here are the account numbers and roles you'll need:

+--------------+----------------+---------------------------+----------------------------------------------------------+
+ Account Name | Account Number | Role Name                 | Role ARN                                                 |
+--------------+----------------+---------------------------+----------------------------------------------------------+
| master       | {{.Organization.MasterAccountId}}   | OrganizationAdministrator | arn:aws:iam::{{.Organization.MasterAccountId}}:role/OrganizationAdministrator |
| audit        | {{.AuditAccount.Id}}   | Auditor                   | arn:aws:iam::{{.AuditAccount.Id}}:role/Auditor                   |
| deploy       | {{.DeployAccount.Id}}   | ?                         | arn:aws:iam::{{.DeployAccount.Id}}:role/?                         |
| network      | {{.NetworkAccount.Id}}   | NetworkAdministrator      | arn:aws:iam::{{.NetworkAccount.Id}}:role/NetworkAdministrator      |
+--------------+----------------+---------------------------+----------------------------------------------------------+

There's also the ops account (account number {{.OpsAccount.Id}}) where you'll land
when you launch a dev/ops EC2 instance or login to the AWS Console via Okta.
`
}
