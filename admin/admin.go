package admin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

func CannedAssumeRolePolicyDocuments(ctx context.Context, cfg *awscfg.Config, bootstrapping bool) (
	canned struct{ AdminRolePrincipals, AuditorRolePrincipals, OrgAccountPrincipals *policies.Document }, // TODO different names without "Principals"?
	err error,
) {
	cp, err := cannedPrincipals(ctx, cfg, bootstrapping)

	var extraAdmin, extraAuditor policies.Document
	if err := jsonutil.Read("substrate.Administrator.assume-role-policy.json", &extraAdmin); err == nil {
		ui.Printf("merging substrate.Administrator.assume-role-policy.json into Administrator's assume role policy")
	} else {
		if errors.Is(err, fs.ErrNotExist) {
			ui.Printf("substrate.Administrator.assume-role-policy.json not found; create it if you wish to customize who can assume Administrator roles")
		} else {
			ui.Printf("error processing substrate.Administrator.assume-role-policy.json: %v", err)
		}
	}
	if err := jsonutil.Read("substrate.Auditor.assume-role-policy.json", &extraAuditor); err == nil {
		ui.Printf("merging substrate.Auditor.assume-role-policy.json into Auditor's assume role policy")
	} else {
		if errors.Is(err, fs.ErrNotExist) {
			ui.Printf("substrate.Auditor.assume-role-policy.json not found; create it if you wish to customize who can assume Auditor roles")
		} else {
			ui.Printf("error processing substrate.Auditor.assume-role-policy.json: %v", err)
		}
	}
	//log.Printf("%+v", extraAdmin)
	//log.Printf("%+v", extraAuditor)

	canned.AdminRolePrincipals = policies.Merge(
		policies.AssumeRolePolicyDocument(cp.AdminRolePrincipals),
		&extraAdmin,
	)
	canned.AuditorRolePrincipals = policies.Merge(
		policies.AssumeRolePolicyDocument(cp.AuditorRolePrincipals),
		&extraAuditor,
	)
	//log.Printf("%+v", canned.AdminRolePrincipals)
	//log.Printf("%+v", canned.AuditorRolePrincipals)

	canned.OrgAccountPrincipals = policies.AssumeRolePolicyDocument(cp.OrgAccountPrincipals)
	return canned, err
}

// EnsureAdminRolesAndPolicies creates or updates the entire matrix of
// Administrator roles and policies to allow the management account and admin
// accounts to move fairly freely throughout the organization.  The given
// cfg must be for the OrganizationAdministrator user or role in the management
// account. If doCloudWatch is true, it'll also reconfigure all the CloudWatch
// cross-account, cross-region roles; this is slow so it's only done when a new
// AWS account's being created since otherwise it's a no-op.
func EnsureAdminRolesAndPolicies(ctx context.Context, cfg *awscfg.Config, doCloudWatch bool) {

	// Gather lists of accounts.  These are used below in configuring policies
	// to allow cross-account access.  On the first run they're basically
	// no-ops but on subsequent runs this is key to not undoing the work of
	// `substrate create-account` and `substrate create-admin-account`.
	canned, err := CannedAssumeRolePolicyDocuments(
		ctx,
		cfg,
		true, // can always be true because we never jump from Intranet to Administrator in other accounts
	)
	if err != nil {
		log.Fatal(err)
	}

	// Admin accounts, once they exist, are going to need to be able to assume
	// a role in the management account.  Because no admin accounts exist yet, this
	// is something of a no-op but it provides a place for them to attach
	// themselves as they're created.
	ui.Spin("finding or creating a role to allow admin accounts to administer your organization")
	role, err := awsiam.EnsureRoleWithPolicy(
		ctx,
		cfg,
		roles.OrganizationAdministrator,
		canned.AdminRolePrincipals,
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
		ctx,
		cfg,
		roles.OrganizationReader,
		canned.OrgAccountPrincipals,
		&policies.Document{
			Statement: []policies.Statement{{
				Action: []string{
					"organizations:DescribeAccount",
					"organizations:DescribeOrganization",
					"organizations:ListAccounts",
					"organizations:ListTagsForResource",
				},
				Resource: []string{"*"},
			}},
		},
	)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)
	if doCloudWatch {
		ui.Spin("finding or creating a role to allow CloudWatch to discover accounts within your organization, too")
		role, err = awsiam.EnsureRoleWithPolicy(
			ctx,
			cfg,
			"CloudWatch-CrossAccountSharing-ListAccountsRole",
			canned.OrgAccountPrincipals,
			&policies.Document{
				Statement: []policies.Statement{{
					Action: []string{
						"organizations:ListAccounts",
						"organizations:ListAccountsForParent",
					},
					Resource: []string{"*"},
				}},
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		ui.Stopf("role %s", role.Name)
		//log.Printf("%+v", role)
	} else {
		ui.Print("not managing CloudWatch cross-account sharing roles because no account was created")
	}

	// Ensure admin accounts can get into the audit account to look at
	// CloudTrail logs.  It's unfortunate that we can't grant admin accounts
	// direct access to the CloudTrail S3 bucket but, since the objects are
	// unavoidably owned by CloudTrail and merely grant access to the bucket
	// owner, bucket policy is powerless to delegate s3:GetObject to the rest
	// of the organization.  Per
	// - <https://aws.amazon.com/premiumsupport/knowledge-center/s3-cross-account-access-denied/>
	// - <https://aws.amazon.com/premiumsupport/knowledge-center/s3-bucket-owner-access/>
	// there is no affordance for making CloudTrail do what I want.  Oh well.
	// This is not simply EnsureAuditorRole because in the audit account we
	// actually do want auditors to be able to read from S3, etc.
	ui.Spin("finding or creating a role to allow admin accounts to audit your organization")
	auditCfg := awscfg.Must(cfg.AssumeSpecialRole(
		ctx,
		accounts.Audit,
		roles.OrganizationAccountAccessRole,
		time.Hour,
	))
	role, err = awsiam.EnsureRole(ctx, auditCfg, roles.Auditor, canned.AuditorRolePrincipals)
	if err != nil {
		ui.Fatal(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, auditCfg, roles.Auditor, "arn:aws:iam::aws:policy/AmazonAthenaFullAccess"); err != nil {
		ui.Fatal(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, auditCfg, roles.Auditor, "arn:aws:iam::aws:policy/ReadOnlyAccess"); err != nil {
		ui.Fatal(err)
	}
	/*
		if err := awsiam.AttachRolePolicy(ctx, auditCfg, roles.Auditor, "arn:aws:iam::aws:policy/SecurityAudit"); err != nil {
			ui.Fatal(err)
		}
	*/
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	// Ensure all admin accounts and the management account can administer the
	// deploy and network accounts.
	ui.Spin("finding or creating a role to allow admin accounts to administer your deploy artifacts")
	deployCfg := awscfg.Must(cfg.AssumeSpecialRole(
		ctx,
		accounts.Deploy,
		roles.OrganizationAccountAccessRole,
		time.Hour,
	))
	role, err = awsiam.EnsureRoleWithPolicy(
		ctx,
		deployCfg,
		roles.DeployAdministrator,
		canned.AdminRolePrincipals,
		&policies.Document{
			Statement: []policies.Statement{{
				Action:   []string{"*"},
				Resource: []string{"*"},
			}},
		},
	)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)
	ui.Spin("finding or creating a role to allow admin accounts to audit your deploy artifacts")
	role, err = EnsureAuditorRole(
		ctx,
		deployCfg,
		canned.OrgAccountPrincipals,
	)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)
	ui.Spin("finding or creating a role to allow admin accounts to administer your networks")
	networkCfg := awscfg.Must(cfg.AssumeSpecialRole(
		ctx,
		accounts.Network,
		roles.OrganizationAccountAccessRole,
		time.Hour,
	))
	role, err = awsiam.EnsureRoleWithPolicy(
		ctx,
		networkCfg,
		roles.NetworkAdministrator,
		canned.AdminRolePrincipals,
		&policies.Document{
			Statement: []policies.Statement{{
				Action:   []string{"*"},
				Resource: []string{"*"},
			}},
		},
	)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)
	ui.Spin("finding or creating a role to allow admin accounts to audit your networks (and Terraform code to discover them)")
	role, err = EnsureAuditorRole(
		ctx,
		networkCfg,
		canned.OrgAccountPrincipals,
	)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	allAccounts, err := awsorgs.ListAccounts(ctx, cfg)
	if err != nil {
		ui.Fatal(err)
	}

	// Loop through every non-admin, non-special account and authorize all the
	// Administrator roles to assume roles there.
	ui.Spinf("finding or creating Administrator and Auditor roles in all other accounts")
	terraformPrincipals := make([]string, 0, len(allAccounts))
	for _, account := range allAccounts {

		// Special accounts have special administrator role names but, still,
		// some of them should be allowed to run Terraform.
		if account.Tags[tags.SubstrateSpecialAccount] != "" {
			switch account.Tags[tags.SubstrateSpecialAccount] {
			case accounts.Deploy:
				terraformPrincipals = append(terraformPrincipals, roles.Arn(aws.ToString(account.Id), roles.DeployAdministrator))
			case accounts.Management:
				terraformPrincipals = append(
					terraformPrincipals,
					roles.Arn(aws.ToString(account.Id), roles.OrganizationAdministrator),
					users.Arn(aws.ToString(account.Id), users.OrganizationAdministrator),
				)
			case accounts.Network:
				terraformPrincipals = append(terraformPrincipals, roles.Arn(aws.ToString(account.Id), roles.NetworkAdministrator))
			}
			continue
		}

		// Every other Substrate-managed account uses the role name
		// "Administrator" and uses it to run Terraform.
		if account.Tags[tags.Domain] != "" {
			terraformPrincipals = append(terraformPrincipals, roles.Arn(aws.ToString(account.Id), roles.Administrator))
		}

		// But the Administrator role in admin accounts is created during IdP
		// configuration so there's nothing more for us to do here.
		if account.Tags[tags.Domain] == "admin" {
			continue
		}

		// In service accounts, though, manage the Administrator and Auditor
		// roles. Try to get in as OrganizationAccountAccessRole, which will
		// exist for accounts created in the organization, but also try to get
		// in as Administrator, to cover accounts invited into the organization
		// that follow the Substrate manual.
		var serviceCfg *awscfg.Config
		for _, roleName := range []string{roles.OrganizationAccountAccessRole} {
			serviceCfg, err = cfg.AssumeRole(
				ctx,
				aws.ToString(account.Id),
				roleName,
				time.Hour,
			)
			if err == nil {
				break
			}
		}
		if err == nil {
			if _, err := EnsureAdministratorRole(ctx, serviceCfg, canned.AdminRolePrincipals); err != nil {
				ui.Fatal(err)
			}
			if _, err := EnsureAuditorRole(ctx, serviceCfg, canned.AuditorRolePrincipals); err != nil {
				ui.Fatal(err)
			}
		} else {
			ui.Printf(
				"could not assume OrganizationAccountAccessRole or Administrator in account %s; not able to manage the Administrator or Auditor roles there; create Administrator per <https://src-bin.com/substrate/manual/getting-started/integrating-your-original-aws-account/> to resolve this warning",
				account.Id,
			)
		}

	}
	ui.Stop("ok")

	if doCloudWatch {
		ui.Spinf("finding or creating the CloudWatch-CrossAccountSharingRole role in all accounts and CloudWatch's service-linked role in admin accounts")
		org, err := cfg.DescribeOrganization(ctx)
		if err != nil {
			ui.Fatal(err)
		}
		for _, account := range allAccounts {
			var accountCfg *awscfg.Config
			if aws.ToString(account.Id) == aws.ToString(org.MasterAccountId) {
				accountCfg = cfg
			} else {
				accountCfg = awscfg.Must(cfg.AssumeRole(
					ctx,
					aws.ToString(account.Id),
					roles.OrganizationAccountAccessRole,
					time.Hour,
				))
			}

			if _, err := EnsureCloudWatchCrossAccountSharingRole(ctx, accountCfg, canned.OrgAccountPrincipals); err != nil {
				ui.Printf(
					"could not create the CloudWatch-CrossAccountSharingRole role in account %s; it might be because this account has only half-joined the organization",
					account.Id,
				)
			}

			if account.Tags[tags.Domain] == "admin" {
				if _, err := awsiam.EnsureServiceLinkedRole(
					ctx,
					accountCfg,
					"AWSServiceRoleForCloudWatchCrossAccount",
					"cloudwatch-crossaccount.amazonaws.com",
				); err != nil {
					ui.Fatal(err)
				}
			}

		}
		ui.Stopf("ok")
	} else {
		ui.Print("not managing CloudWatch cross-account sharing roles because no account was created")
	}

	// Ensure every account can run Terraform with remote state centralized
	// in the deploy account.
	ui.Spin("finding or creating an IAM role for Terraform to use to manage remote state")
	sort.Strings(terraformPrincipals) // to avoid spurious policy diffs
	//log.Printf("%+v", terraformPrincipals)
	var resources []string
	for _, region := range regions.All() { // we can't use regions.Selected() in the first `substrate bootstrap-management-account`
		bucketName := terraform.S3BucketName(region)
		resources = append(
			resources,
			fmt.Sprintf("arn:aws:s3:::%s", bucketName),
			fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
		)
	}
	role, err = awsiam.EnsureRoleWithPolicy(
		ctx,
		deployCfg,
		roles.TerraformStateManager,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: terraformPrincipals}),
		&policies.Document{
			Statement: []policies.Statement{
				{
					Action: []string{"dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:DeleteItem"},
					Resource: []string{
						fmt.Sprintf("arn:aws:dynamodb:*:*:table/%s", terraform.DynamoDBTableName),
					},
				},
				{
					Action:   []string{"s3:DeleteObject", "s3:GetObject", "s3:ListBucket", "s3:PutObject"},
					Resource: resources,
				},
			},
		},
	)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

}

// EnsureAdministratorRole creates the Administrator role in the account
// referenced by the given IAM client.  The only restrictions on the APIs
// this role may call are set by the organization's service control policies.
func EnsureAdministratorRole(ctx context.Context, cfg *awscfg.Config, assumeRolePolicyDocument *policies.Document) (*awsiam.Role, error) {
	return awsiam.EnsureRoleWithPolicy(
		ctx,
		cfg,
		roles.Administrator,
		assumeRolePolicyDocument,
		&policies.Document{
			Statement: []policies.Statement{{
				Action:   []string{"*"},
				Resource: []string{"*"},
			}},
		},
	)
}

// EnsureAuditorRole creates the Auditor role in the account referenced by the
// given IAM client.  This role will be allowed to call all read-only APIs as
// defined by the AWS-managed ReadOnlyAccess policy except the list of
// sensitive read-only APIs identified by Alestic and captured in
// DenySensitiveReadsPolicyDocument. It will also be allowed to assume roles
// via AllowAssumeRolePolicyDocument but the roles that allows it to assume
// them will (presumably) also be read-only Auditor-like roles.
func EnsureAuditorRole(ctx context.Context, cfg *awscfg.Config, assumeRolePolicyDocument *policies.Document) (*awsiam.Role, error) {
	role, err := awsiam.EnsureRoleWithPolicy(
		ctx,
		cfg,
		roles.Auditor,
		assumeRolePolicyDocument,
		policies.Merge(
			AllowAssumeRolePolicyDocument, // TODO set a permissions boundary to keep it read-only even if the role that allows Auditor to assume it is more permissive
			DenySensitiveReadsPolicyDocument,
		),
	)
	if err != nil {
		return nil, err
	}
	if err := awsiam.AttachRolePolicy(
		ctx,
		cfg,
		roles.Auditor,
		"arn:aws:iam::aws:policy/ReadOnlyAccess",
	); err != nil {
		return nil, err
	}
	/*
		if err := awsiam.AttachRolePolicy(
			ctx,
			cfg,
			roles.Auditor,
			"arn:aws:iam::aws:policy/SecurityAudit",
		); err != nil {
			return nil, err
		}
	*/
	return role, nil
}

// EnsureCloudWatchCrossAccountSharingRole creates the
// CloudWatch-CrossAccountSharingRole role in the account referenced by the
// given IAM client.  This role will be allowed to read CloudWatch logs and
// metrics via a service-linked role in the admin account(s).
func EnsureCloudWatchCrossAccountSharingRole(
	ctx context.Context,
	cfg *awscfg.Config,
	assumeRolePolicyDocument *policies.Document,
) (*awsiam.Role, error) {
	const roleName = "CloudWatch-CrossAccountSharingRole"
	role, err := awsiam.EnsureRole(
		ctx,
		cfg,
		roleName,
		assumeRolePolicyDocument,
	)
	if err != nil {
		return nil, err
	}

	if err := awsiam.AttachRolePolicy(ctx, cfg, roleName, "arn:aws:iam::aws:policy/job-function/ViewOnlyAccess"); err != nil {
		return nil, err
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, roleName, "arn:aws:iam::aws:policy/AWSXrayReadOnlyAccess"); err != nil {
		return nil, err
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, roleName, "arn:aws:iam::aws:policy/CloudWatchReadOnlyAccess"); err != nil {
		return nil, err
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, roleName, "arn:aws:iam::aws:policy/CloudWatchAutomaticDashboardsAccess"); err != nil {
		return nil, err
	}

	return role, nil
}

func cannedPrincipals(ctx context.Context, cfg *awscfg.Config, bootstrapping bool) (
	canned struct{ AdminRolePrincipals, AuditorRolePrincipals, OrgAccountPrincipals *policies.Principal },
	err error,
) {
	var adminAccounts, allAccounts []*awsorgs.Account
	if adminAccounts, err = cfg.FindAccounts(ctx, func(a *awscfg.Account) bool {
		return a.Tags[tags.Domain] == naming.Admin
	}); err != nil {
		return
	}
	if allAccounts, err = cfg.FindAccounts(ctx, func(*awscfg.Account) bool { return true }); err != nil {
		return
	}
	var org *awscfg.Organization
	org, err = cfg.DescribeOrganization(ctx)
	if err != nil {
		return
	}

	if bootstrapping {
		canned.AdminRolePrincipals = &policies.Principal{AWS: make([]string, len(adminAccounts)+2)}     // +2 for the management account and its IAM user
		canned.AuditorRolePrincipals = &policies.Principal{AWS: make([]string, len(adminAccounts)*2+2)} // *2 for Administrator AND Auditor; +2 for the management account and its IAM user
		for i, account := range adminAccounts {
			canned.AdminRolePrincipals.AWS[i] = roles.Arn(aws.ToString(account.Id), roles.Administrator)
			canned.AuditorRolePrincipals.AWS[i*2] = roles.Arn(aws.ToString(account.Id), roles.Administrator)
			canned.AuditorRolePrincipals.AWS[i*2+1] = roles.Arn(aws.ToString(account.Id), roles.Auditor)
		}
	} else {
		canned.AdminRolePrincipals = &policies.Principal{AWS: make([]string, len(adminAccounts)*2+2)}   // *2 for Administrator AND Intranet; +2 for the management account and its IAM user
		canned.AuditorRolePrincipals = &policies.Principal{AWS: make([]string, len(adminAccounts)*3+2)} // *3 for Administrator AND Auditor AND Intranet; +2 for the management account and its IAM user
		for i, account := range adminAccounts {
			canned.AdminRolePrincipals.AWS[i*2] = roles.Arn(aws.ToString(account.Id), roles.Administrator)
			canned.AdminRolePrincipals.AWS[i*2+1] = roles.Arn(aws.ToString(account.Id), roles.Intranet)
			canned.AuditorRolePrincipals.AWS[i*3] = roles.Arn(aws.ToString(account.Id), roles.Administrator)
			canned.AuditorRolePrincipals.AWS[i*3+1] = roles.Arn(aws.ToString(account.Id), roles.Auditor)
			canned.AuditorRolePrincipals.AWS[i*3+2] = roles.Arn(aws.ToString(account.Id), roles.Intranet)
		}
	}
	canned.AdminRolePrincipals.AWS[len(canned.AdminRolePrincipals.AWS)-2] = roles.Arn(
		aws.ToString(org.MasterAccountId),
		roles.OrganizationAdministrator,
	)
	canned.AdminRolePrincipals.AWS[len(canned.AdminRolePrincipals.AWS)-1] = users.Arn(
		aws.ToString(org.MasterAccountId),
		users.OrganizationAdministrator,
	)
	canned.AuditorRolePrincipals.AWS[len(canned.AuditorRolePrincipals.AWS)-2] = roles.Arn(
		aws.ToString(org.MasterAccountId),
		roles.OrganizationAdministrator,
	)
	canned.AuditorRolePrincipals.AWS[len(canned.AuditorRolePrincipals.AWS)-1] = users.Arn(
		aws.ToString(org.MasterAccountId),
		users.OrganizationAdministrator,
	)
	sort.Strings(canned.AdminRolePrincipals.AWS)   // to avoid spurious policy diffs
	sort.Strings(canned.AuditorRolePrincipals.AWS) // to avoid spurious policy diffs

	canned.OrgAccountPrincipals = &policies.Principal{AWS: make([]string, len(allAccounts))}
	for i, account := range allAccounts {
		canned.OrgAccountPrincipals.AWS[i] = aws.ToString(account.Id)
	}
	sort.Strings(canned.OrgAccountPrincipals.AWS) // to avoid spurious policy diffs

	//log.Printf("%+v", canned)
	return
}
