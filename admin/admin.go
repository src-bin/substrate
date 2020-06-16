package admin

import (
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

// EnsureAdministratorRolesAndPolicies creates or updates the entire matrix of
// Administrator roles and policies to allow the master account and admin
// accounts to move fairly freely throughout the organization.  The given
// session must be for the OrganizationAdministrator user or role in the master
// account.
func EnsureAdministratorRolesAndPolicies(sess *session.Session) error {
	svc := organizations.New(sess)

	org, err := awsorgs.DescribeOrganization(svc)
	if err != nil {
		log.Fatal(err)
	}

	// Gather lists of accounts.  These are used below in configuring policies
	// to allow cross-account access.  On the first run they're basically
	// no-ops but on subsequent runs this is key to not undoing the work of
	// substrate-create-account and substrate-create-admin-account.
	adminAccounts, err := awsorgs.FindAccountsByDomain(svc, accounts.Admin)
	if err != nil {
		log.Fatal(err)
	}
	adminPrincipals := make([]string, len(adminAccounts)+2) // +2 for the master account and its IAM user
	for i, account := range adminAccounts {
		adminPrincipals[i] = roles.Arn(aws.StringValue(account.Id), roles.Administrator)
	}
	adminPrincipals[len(adminPrincipals)-2] = aws.StringValue(org.MasterAccountId)
	adminPrincipals[len(adminPrincipals)-1] = fmt.Sprintf(
		"arn:aws:iam::%s:user/%s",
		aws.StringValue(org.MasterAccountId),
		roles.OrganizationAdministrator,
	)
	sort.Strings(adminPrincipals) // to avoid spurious policy diffs
	//log.Printf("%+v", adminPrincipals)
	allAccounts, err := awsorgs.ListAccounts(svc)
	if err != nil {
		log.Fatal(err)
	}
	allAccountIds := make([]string, len(allAccounts))
	for i, account := range allAccounts {
		allAccountIds[i] = aws.StringValue(account.Id)
	}
	sort.Strings(allAccountIds) // to avoid spurious policy diffs
	//log.Printf("%+v", allAccountIds)

	// Admin accounts, once they exist, are going to need to be able to assume
	// a role in the master account.  Because no admin accounts exist yet, this
	// is something of a no-op but it provides a place for them to attach
	// themselves as they're created.
	ui.Spin("finding or creating a role to allow admin accounts to administer your organization")
	role, err := awsiam.EnsureRoleWithPolicy(
		iam.New(sess),
		roles.OrganizationAdministrator,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: adminPrincipals}),
		&policies.Document{
			Statement: []policies.Statement{{
				Action:   []string{"*"},
				Resource: []string{"*"},
			}},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	// Ensure admin accounts can find other accounts in the organization without full administrative
	// privileges.  This should be directly possible via organizations:RegisterDelegatedAdministrator
	// but that API appears to just not work the way
	// <https://docs.aws.amazon.com/organizations/latest/userguide/orgs_integrated-services-list.html>
	// implies it does.  As above, this is almost a no-op for now because, while it grants access to
	// the special accounts, no admin accounts exist yet.
	ui.Spin("finding or creating a role to allow account discovery within your organization")
	role, err = awsiam.EnsureRoleWithPolicy(
		iam.New(sess),
		roles.OrganizationReader,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: allAccountIds}),
		&policies.Document{
			Statement: []policies.Statement{{
				Action:   []string{"organizations:DescribeOrganization", "organizations:ListAccounts"},
				Resource: []string{"*"},
			}},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	// Ensure the master account can get into the deploy and network accounts
	// the same way admin accounts will in the future.  This will make deploy
	// and network account boostrapping (the next steps) easier.
	ui.Spin("finding or creating a role to allow admin accounts to administer your deploy artifacts")
	deployAccount, err := awsorgs.FindSpecialAccount(svc, accounts.Deploy)
	if err != nil {
		log.Fatal(err)
	}
	role, err = awsiam.EnsureRoleWithPolicy(
		iam.New(awssessions.AssumeRole(
			sess,
			aws.StringValue(deployAccount.Id),
			roles.OrganizationAccountAccessRole,
		)),
		roles.DeployAdministrator,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: adminPrincipals}),
		&policies.Document{
			Statement: []policies.Statement{{
				Action:   []string{"*"},
				Resource: []string{"*"},
			}},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)
	ui.Spin("finding or creating a role to allow admin accounts to administer your networks")
	networkAccount, err := awsorgs.FindSpecialAccount(svc, accounts.Network)
	if err != nil {
		log.Fatal(err)
	}
	role, err = awsiam.EnsureRoleWithPolicy(
		iam.New(awssessions.AssumeRole(
			sess,
			aws.StringValue(networkAccount.Id),
			roles.OrganizationAccountAccessRole,
		)),
		roles.NetworkAdministrator,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: adminPrincipals}),
		&policies.Document{
			Statement: []policies.Statement{{
				Action:   []string{"*"},
				Resource: []string{"*"},
			}},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	return nil
}
