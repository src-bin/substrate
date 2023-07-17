package createrole

import (
	"context"
	"flag"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	roleName := flag.String("role", "", "name of the IAM role to create")
	selectionFlags := accounts.NewSelectionFlags(accounts.SelectionFlagsUsage{
		AllDomains:      "create the role in all domains (potentially constrained by -environment and/or -quality)",
		Domains:         "only create this role in AWS accounts in this domain (may be repeated)",
		AllEnvironments: "create the role in all environments (potentially constrained by -domain and/or -quality)",
		Environments:    "only create this role in AWS accounts in this environment (may be repeated)",
		AllQualities:    "create the role in all qualities (potentially constrained by -domain and/or -environment)",
		Qualities:       "only create this role in AWS accounts of this quality (may be repeated)",
		Admin:           "create this role in the organization's admin account(s) (potentially constrained by -quality)",
		Management:      "create this role in the organization's management AWS account",
		Specials:        `create this role in a special AWS account (may be repeated; "deploy" and/or "network")`,
		Numbers:         "create this role in a specific AWS account, by 12-digit account number (may be repeated)",
	})
	managedAssumeRolePolicyFlags := roles.NewManagedAssumeRolePolicyFlags(roles.ManagedAssumeRolePolicyFlagsUsage{
		Humans:        "allow humans with this role set in your IdP to assume this role via the Credential Factory (implies -admin)",
		AWSServices:   `allow an AWS service (by URL; e.g. "ec2.amazonaws.com") to assume role (may be repeated)`,
		GitHubActions: `allow GitHub Actions to assume this role in the context of the given GitHub organization and repository (separated by a literal '/'; may be repeated)`,
		Filenames:     "filename containing an assume-role policy to be merged into this role's final assume-role policy (may be repeated)",
	})
	managedPolicyAttachmentsFlags := roles.NewManagedPolicyAttachmentsFlags(roles.ManagedPolicyAttachmentsFlagsUsage{
		AdministratorAccess: "attach the AWS-managed AdministratorAccess policy to these roles, allowing total access to all AWS APIs and resources",
		ReadOnlyAccess:      "attach the AWS-managed ReadOnlyAccess policy to these roles, allowing read access to all AWS resources",
		ARNs:                "attach a specific AWS-managed policy to these roles (may be repeated)",
		Filenames:           "filename containing a policy to attach to these roles (may be repeated)",
	})
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	flag.Usage = func() {
		ui.Print("Usage: substrate create-role [account selection flags] -role <role> [assume-role policy flags] [policy attachment flags] [-quiet]")
		ui.Print("       [account selection flags]:  [-all-domains|-domain <domain> [...]]")
		ui.Print("                                   [-all-environments|-environment <environment> [...]]")
		ui.Print("                                   [-all-qualities|-quality <quality> [...]]")
		ui.Print("                                   [-admin] [-management] [-special <special> [...]]")
		ui.Print("                                   [-number <number> [...]]")
		ui.Print("       [assume-role policy flags]: [-humans] [-aws-service <aws-service-url>] [-github-actions <org/repo>] [-assume-role-policy <filename> [...]]")
		ui.Print("       [policy attachment flags]:  [-administrator-access|-read-only-access] [-policy-arn <arn> [...]] [-policy <filename> [...]]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *quiet {
		ui.Quiet()
	}
	if *roleName == "" {
		ui.Fatal(`-role "..." is required`)
	}
	if *roleName == roles.Administrator || *roleName == roles.Auditor {
		ui.Fatalf("cannot manage %q with `substrate create-role`", *roleName)
	}
	managedAssumeRolePolicy, err := managedAssumeRolePolicyFlags.ManagedAssumeRolePolicy()
	ui.Must(err)
	//log.Printf("%+v", managedAssumeRolePolicy)
	managedPolicyAttachments, err := managedPolicyAttachmentsFlags.ManagedPolicyAttachments()
	ui.Must(err)
	//log.Printf("%+v", managedPolicyAttachments)
	selection, err := selectionFlags.Selection()
	ui.Must(err)
	if managedAssumeRolePolicy.Humans {
		selection.Humans = true
	}
	//log.Printf("%+v", selection)

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	versionutil.PreventDowngrade(ctx, cfg)

	// Partition accounts by the given options so the role may be created or
	// deleted as appropriate.
	ui.Spin("inspecting all your AWS accounts")
	selected, unselected, err := selection.Partition(ctx, cfg)
	ui.Must(err)
	ui.Stop("ok")

	// Delete this role in accounts where it's no longer necessary per the
	// given options. We do this first so that if one of the confirmations
	// spooks the user, there's less to unwind.
	if len(unselected) > 0 {
		ui.Printf("finding Substrate-managed %s roles that should now be deleted according to these account selection flags", *roleName)
		for _, account := range unselected {
			if err := awsiam.DeleteRoleWithConfirmation(
				ctx,
				awscfg.Must(account.Config(
					ctx,
					cfg,
					account.AdministratorRoleName(),
					time.Hour,
				)),
				*roleName,
				false, // always confirm these probably surprising deletes
			); err != nil && !awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
				ui.Fatal(err)
			}
		}
	}

	// Every role we create needs these minimal privileges in order to use
	// the Substrate command-line tools.
	minimalPolicy := &policies.Document{
		Statement: []policies.Statement{{
			Action: []string{
				"organizations:DescribeOrganization",
				"sts:AssumeRole",
			},
			Resource: []string{"*"},
		}},
	}

	// We'll also need the canned principals we use in the various
	// Administrator roles, too, so that they can have an on-ramp into the web
	// of roles we're creating here, too.
	canned, err := admin.CannedPrincipals(ctx, cfg, false)
	ui.Must(err)

	// If this role's for humans to use via the IdP, create a role by the same
	// name in admin accounts. This role must exist before we enter the main
	// role and policy loop because that loop will need to reference these role
	// ARNs and they must exist at that time. If -admin was given in addition
	// to -humans, we'll use CreateRole here and suppress EntityAlreadyExists,
	// knowing that this role will be thoroughly managed later and preventing
	// a momentary regression in the assume-role policy. If -humans was given
	// without -admin then this is our only shot at managing this role so we'll
	// use EnsureRoleWithPolicy.
	adminPrincipals := &policies.Principal{AWS: []string{}}
	if selection.Humans {
		ui.Spinf("finding or creating the %s role in your Substrate and admin account(s) for humans to assume via your IdP", *roleName)
		adminAccounts, _, substrateAccount, _, _, _, _, err := accounts.Grouped(ctx, cfg)
		ui.Must(err)

		for _, account := range append(adminAccounts, substrateAccount) {
			accountCfg := awscfg.Must(account.Config(ctx, cfg, roles.Administrator, time.Hour))
			var role *awsiam.Role
			if selection.Admin {
				role, err = awsiam.CreateRole(
					ctx,
					accountCfg,
					*roleName,
					policies.AssumeRolePolicyDocument(canned.AdminRolePrincipals),
				)
			} else {
				role, err = awsiam.EnsureRoleWithPolicy(
					ctx,
					accountCfg,
					*roleName,
					policies.AssumeRolePolicyDocument(canned.AdminRolePrincipals),
					minimalPolicy,
				)
			}
			if err == nil {
				ui.Must(awsiam.TagRole(ctx, cfg, role.Name, tagging.Map{
					tagging.SubstrateAccountSelectors: "humans",
				}))
			} else if awsutil.ErrorCodeIs(err, awsiam.EntityAlreadyExists) {
				role, err = awsiam.GetRole(ctx, accountCfg, *roleName)
			}
			ui.Must(err)
			adminPrincipals.AWS = append(adminPrincipals.AWS, role.ARN)
		}
		ui.Stop("ok")
	}

	// Create the role in each account, constructing the assume-role policy
	// uniquely for each one because certain aspects, e.g. the GitHub Actions
	// OAuth OIDC provider, may differ in the details from one account to
	// another.
	for _, as := range selected {
		account := as.Account
		accountCfg := awscfg.Must(account.Config(ctx, cfg, account.AdministratorRoleName(), time.Hour))
		selectors := as.Selectors
		ui.Printf("constructing an assume-role policy for the %s role in %s", *roleName, account)

		// Start the assume-role policy for the role in this account with the
		// standard Substrate-managed principals like Administrator and
		// OrganizationAdministrator.
		assumeRolePolicy := policies.AssumeRolePolicyDocument(canned.AdminRolePrincipals) // Administrator can do anything, after all

		// If -humans was given, allow the pre-created roles in the admin
		// account(s) to assume this role.
		if managedAssumeRolePolicy.Humans {
			ui.Printf("allowing humans to assume the %s role in %s via admin accounts and your IdP", *roleName, account)
			assumeRolePolicy = policies.Merge(
				assumeRolePolicy,
				policies.AssumeRolePolicyDocument(adminPrincipals),
			)

			// Further, if this account is, in fact, an admin account or the
			// Substrate account, allow the Intranet's and the Substrate user's
			// principals to assume the role, too, so the Credential Factory
			// will work and create an EC2 instance profile so the Instance
			// Factory can use the role.
			if account.Tags[tagging.Domain] == naming.Admin {
				assumeRolePolicy = policies.Merge(
					assumeRolePolicy,
					policies.AssumeRolePolicyDocument(&policies.Principal{
						AWS: []string{
							roles.ARN(aws.ToString(account.Id), roles.Intranet),
							users.ARN(aws.ToString(account.Id), users.CredentialFactory),
						},
						Service: []string{"ec2.amazonaws.com"},
					}),
				)
				_, err = awsiam.EnsureInstanceProfile(ctx, cfg, *roleName)
				ui.Must(err)
			}

		}

		if len(managedAssumeRolePolicy.AWSServices) > 0 {
			ui.Printf("allowing %s to assume the %s role in %s", strings.Join(managedAssumeRolePolicy.AWSServices, ", "), *roleName, account)
			assumeRolePolicy = policies.Merge(
				assumeRolePolicy,
				policies.AssumeRolePolicyDocument(&policies.Principal{
					Service: jsonutil.StringSlice(managedAssumeRolePolicy.AWSServices),
				}),
			)
		}

		if len(managedAssumeRolePolicy.GitHubActions) > 0 {
			ui.Printf(
				"allowing GitHub Actions to assume the %s role in %s on behalf of %s",
				*roleName,
				account,
				strings.Join(managedAssumeRolePolicy.GitHubActions, ", "),
			)
			arn, err := awsiam.EnsureOpenIDConnectProvider(
				ctx,
				accountCfg,
				[]string{"sts.amazonaws.com"},
				[]string{awsiam.GitHubActionsOAuthOIDCThumbprint},
				awsiam.GitHubActionsOAuthOIDCURL,
			)
			ui.Must(err)
			subs, err := managedAssumeRolePolicy.GitHubActionsSubs()
			ui.Must(err)
			assumeRolePolicy = policies.Merge(
				assumeRolePolicy,
				&policies.Document{
					Statement: []policies.Statement{{
						Action: []string{"sts:AssumeRoleWithWebIdentity"},
						Condition: policies.Condition{"StringEquals": {
							"token.actions.githubusercontent.com:sub": subs,
						}},
						Principal: &policies.Principal{
							Federated: []string{arn},
						},
					}},
				},
			)
		}

		for _, filename := range managedAssumeRolePolicy.Filenames {
			ui.Printf("reading additional assume-role policy statements from %s", filename)
			var filePolicy policies.Document
			ui.Must(jsonutil.Read(filename, &filePolicy))
			assumeRolePolicy = policies.Merge(assumeRolePolicy, &filePolicy)
		}

		ui.Spinf("finding or creating the %s role in %s", *roleName, account)
		role, err := awsiam.EnsureRoleWithPolicy(
			ctx,
			accountCfg,
			*roleName,
			assumeRolePolicy,
			minimalPolicy,
		)
		ui.Must(err)
		tags := tagging.Map{
			tagging.SubstrateAccountSelectors: strings.Join(selectors, " "),
		}
		if len(managedAssumeRolePolicy.Filenames) > 0 {
			tags[tagging.SubstrateAssumeRolePolicyFilenames] = strings.Join(managedAssumeRolePolicy.Filenames, " ")
		}
		if len(managedPolicyAttachments.Filenames) > 0 {
			tags[tagging.SubstratePolicyAttachmentFilenames] = strings.Join(managedPolicyAttachments.Filenames, " ")
		}
		ui.Must(awsiam.TagRole(ctx, accountCfg, role.Name, tags))
		for _, service := range managedAssumeRolePolicy.AWSServices {
			if service == "ec2.amazonaws.com" {
				_, err = awsiam.EnsureInstanceProfile(ctx, accountCfg, *roleName)
				ui.Must(err)
			}
		}
		ui.Stopf("ok")

		// Attach policies to selected accounts. If this account is an admin
		// account, only attach policies if -admin was given. If this account
		// is not an admin account, attach them without further conditions.
		// This complication is because -humans implies that roles must exist
		// in admin accounts but not that they should have all the same
		// permissions in admin accounts that they have elsewhere.
		if isAdminAccount := account.Tags[tagging.Domain] == naming.Admin; isAdminAccount && selection.Admin || !isAdminAccount {
			if managedPolicyAttachments.AdministratorAccess {
				ui.Spinf("attaching the AdministratorAccess policy to the %s role in %s", *roleName, account)
				ui.Must(awsiam.AttachRolePolicy(
					ctx,
					accountCfg,
					*roleName,
					"arn:aws:iam::aws:policy/AdministratorAccess",
				))
				ui.Stopf("ok")
			}
			if managedPolicyAttachments.ReadOnlyAccess {
				ui.Spinf("attaching the ReadOnlyAccess policy to the %s role in %s", *roleName, account)
				ui.Must(awsiam.AttachRolePolicy(
					ctx,
					accountCfg,
					*roleName,
					"arn:aws:iam::aws:policy/ReadOnlyAccess",
				))
				ui.Stopf("ok")
			}
			if len(managedPolicyAttachments.ARNs) > 0 {
				ui.Spinf("attaching AWS-managed policies to the %s role in %s", *roleName, account)
				for _, arn := range managedPolicyAttachments.ARNs {
					ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, *roleName, arn))
				}
				ui.Stopf("ok")
			}
			if len(managedPolicyAttachments.Filenames) > 0 {
				ui.Spinf("merging custom policies into the %s role in %s", *roleName, account)
				policy := minimalPolicy
				for _, filename := range managedPolicyAttachments.Filenames {
					var filePolicy policies.Document
					ui.Must(jsonutil.Read(filename, &filePolicy))
					policy = policies.Merge(policy, &filePolicy)
				}
				ui.Must(awsiam.PutRolePolicy(ctx, accountCfg, *roleName, awsiam.SubstrateManaged, policy))
				ui.Stopf("ok")
			}
		}

	}

}
