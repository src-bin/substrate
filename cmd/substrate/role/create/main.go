package create

import (
	"context"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/humans"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

var (
	roleName                 = new(string)
	selection                = &accounts.Selection{}
	managedAssumeRolePolicy  = &roles.ManagedAssumeRolePolicy{}
	managedPolicyAttachments = &roles.ManagedPolicyAttachments{}
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: `create [account selection flags] --role <role> [assume-role policy flags] [policy attachment flags] [--quiet]
    [account selection flags]:  [--all-domains|--domain <domain> [...]]
                                [--all-environments|--environment <environment> [...]]
                                [--all-qualities|--quality <quality> [...]]
                                [--management] [--special <special> [...]] [--substrate]
                                [--number <number> [...]]
    [assume-role policy flags]: [--humans]
                                [--aws-service <service.amazonaws.com>] [--github-actions <org/repo>]
                                [--assume-role-policy <filename> [...]]
    [policy attachment flags]:  [--administrator-access|--read-only-access]
                                [--policy-arn <arn> [...]] [--policy <filename> [...]]`,
		Short: "create or update an AWS IAM role in selected AWS accounts",
		Long:  ``,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--role",
				"--all-domains", "--domain",
				"--all-environments", "--environment",
				"--all-qualities", "--quality",
				"--management", "--special", "--substrate",
				"--number",
				"--humans", "--aws-service", "--github-actions", "--assume-role-policy",
				"--administrator-access", "--read-only-access", "--policy-arn", "--policy",
				"--quiet",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().StringVar(roleName, "role", "", "name of the IAM role to create")
	cmd.RegisterFlagCompletionFunc("role", cmdutil.NoCompletionFunc)
	cmd.Flags().AddFlagSet(selection.FlagSet(accounts.SelectionFlagsUsage{
		AllDomains:      "create the role in all domains (potentially constrained by --environment and/or --quality)",
		Domains:         "only create this role in AWS accounts in this domain (may be repeated)",
		AllEnvironments: "create the role in all environments (potentially constrained by --domain and/or --quality)",
		Environments:    "only create this role in AWS accounts in this environment (may be repeated)",
		AllQualities:    "create the role in all qualities (potentially constrained by --domain and/or --environment)",
		Qualities:       "only create this role in AWS accounts of this quality (may be repeated)",
		Substrate:       "create this role in the organization's Substrate account",
		Management:      "create this role in the organization's management AWS account",
		Specials:        `create this role in a special AWS account (may be repeated; "audit", "deploy", and/or "network")`,
		Numbers:         "create this role in a specific AWS account, by 12-digit account number (may be repeated)",
	}))
	cmd.Flags().AddFlagSet(managedAssumeRolePolicy.FlagSet(roles.ManagedAssumeRolePolicyFlagsUsage{
		Humans:        "allow humans with this role set in your IdP to assume this role via the Credential Factory (implies --substrate)",
		AWSServices:   `allow an AWS service (by URL; e.g. "ec2.amazonaws.com") to assume role (may be repeated)`,
		GitHubActions: `allow GitHub Actions to assume this role in the context of the given GitHub organization and repository (separated by a literal '/'; may be repeated)`,
		Filenames:     "filename containing an assume-role policy to be merged into this role's final assume-role policy (may be repeated)",
	}))
	cmd.Flags().AddFlagSet(managedPolicyAttachments.FlagSet(roles.ManagedPolicyAttachmentsFlagsUsage{
		AdministratorAccess: "attach the AWS-managed AdministratorAccess policy to these roles, allowing total access to all AWS APIs and resources",
		ReadOnlyAccess:      "attach the AWS-managed ReadOnlyAccess policy to these roles, allowing read access to all AWS resources",
		ARNs:                "attach a specific AWS-managed policy to these roles (may be repeated)",
		Filenames:           "filename containing a policy to attach to these roles (may be repeated)",
	}))
	cmd.Flags().AddFlag(cmdutil.QuietFlag())
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {
	if *roleName == "" {
		ui.Fatal(`--role "..." is required`)
	}
	if *roleName == roles.Administrator || *roleName == roles.Auditor {
		ui.Fatalf("cannot manage %s roles with `substrate role create`", *roleName)
	}
	ui.Must(managedAssumeRolePolicy.Validate())
	//log.Printf("%+v", managedAssumeRolePolicy)
	ui.Must(managedPolicyAttachments.Validate())
	//log.Printf("%+v", managedPolicyAttachments)
	ui.Must(selection.Validate())
	if managedAssumeRolePolicy.Humans {
		selection.Humans = true
	}
	//log.Printf("%+v", selection)
	//log.Print(jsonutil.MustString(selection))

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

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

	// And during the transition from (theoretically multiple) admin accounts
	// to exactly one Substrate account, we need to know about those beyond
	// the `if selection.Humans` block below here.
	adminAccounts, _, substrateAccount, _, _, _, _, err := accounts.Grouped(ctx, cfg)
	ui.Must(err)

	intranetAssumeRolePolicy, err := humans.IntranetAssumeRolePolicy(ctx, cfg)
	ui.Must(err)

	// If this role's for humans to use via the IdP, create a role by the same
	// name in the Substrate account. This role must exist before we enter the
	// main role and policy loop because that loop will need to reference these
	// role ARNs and they must exist at that time. If -admin was given in
	// addition to -humans, we'll use CreateRole here and suppress
	// EntityAlreadyExists, knowing that this role will be thoroughly managed
	// later and preventing a momentary regression in the assume-role policy.
	// If -humans was given without -substrate then this is our only shot at
	// managing this role so we'll use EnsureRoleWithPolicy.
	adminPrincipals := &policies.Principal{AWS: []string{}} // TODO turn into a singular substratePrincipal when removing admin accounts
	if selection.Humans {
		ui.Spinf("finding or creating the %s role in your Substrate account for humans to assume via your IdP", *roleName)

		for _, account := range append(adminAccounts, substrateAccount) {
			if account == nil { // substrateAccount will be nil until they've run `substrate setup`
				continue
			}
			accountCfg := awscfg.Must(account.Config(ctx, cfg, roles.Administrator, time.Hour))
			var role *awsiam.Role
			if selection.Substrate {
				role, err = awsiam.CreateRole(
					ctx,
					accountCfg,
					*roleName,
					intranetAssumeRolePolicy,
				)
			} else {
				role, err = awsiam.EnsureRoleWithPolicy(
					ctx,
					accountCfg,
					*roleName,
					intranetAssumeRolePolicy,
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

		// Begin with an empty (technically nil) assume-role policy.
		var assumeRolePolicy *policies.Document

		// If -humans was given, allow the pre-created roles in the Substrate
		// account to assume this role. This is much simpler than it used to be
		// because the Intranet's principals now assume roles directly in other
		// accounts to allow 12-hour sessions and we're now creating EC2
		// instance profiles everywhere in anticipation of Instance Factory
		// being able to launch instances in any account (as oft requested).
		if managedAssumeRolePolicy.Humans {
			ui.Printf("allowing humans to assume the %s role in %s via your IdP", *roleName, account)
			assumeRolePolicy = policies.Merge(
				assumeRolePolicy,
				intranetAssumeRolePolicy,
				policies.AssumeRolePolicyDocument(adminPrincipals),
			)
			ui.Must2(awsiam.EnsureInstanceProfile(ctx, cfg, *roleName))
		}

		if len(managedAssumeRolePolicy.AWSServices) > 0 {
			ui.Printf(
				"allowing %s to assume the %s role in %s",
				strings.Join(managedAssumeRolePolicy.AWSServices, ", "),
				*roleName,
				account,
			)
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

		if assumeRolePolicy == nil {
			ui.Fatal("at least one assume-role policy flag is required")
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
		// account or the Substrate account, only attach policies if -admin
		// or -substrate was given. If this account is not an admin account or
		// the Substrate account, attach them without further conditions.
		// This complication is because -humans implies that roles must exist
		// in admin/Substrate accounts but not that they should have all the
		// same permissions in those accounts that they have elsewhere.
		if is := account.Tags[tagging.Domain] == naming.Admin || account.Tags[tagging.SubstrateType] == naming.Substrate; is && selection.Substrate || !is {

			if managedPolicyAttachments.AdministratorAccess {
				ui.Spinf("attaching the AdministratorAccess policy to the %s role in %s", *roleName, account)
			}
			if managedPolicyAttachments.ReadOnlyAccess {
				ui.Spinf("attaching the ReadOnlyAccess policy to the %s role in %s", *roleName, account)
			}
			if managedPolicyAttachments.AdministratorAccess {
				ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, *roleName, policies.AdministratorAccess))
			} else {
				ui.Must(awsiam.DetachRolePolicy(ctx, accountCfg, *roleName, policies.AdministratorAccess))
			}
			if managedPolicyAttachments.ReadOnlyAccess {
				ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, *roleName, policies.ReadOnlyAccess))
			} else {
				ui.Must(awsiam.DetachRolePolicy(ctx, accountCfg, *roleName, policies.ReadOnlyAccess))
			}
			if managedPolicyAttachments.AdministratorAccess || managedPolicyAttachments.ReadOnlyAccess {
				ui.Stop("ok")
			}

			attachedARNs := ui.Must2(awsiam.ListAttachedRolePolicies(ctx, accountCfg, *roleName))
			managedPolicyAttachments.Sort()
			arns := managedPolicyAttachments.ARNs // just to shorten its name in the loop below
			for _, arn := range attachedARNs {
				if arn == policies.AdministratorAccess || arn == policies.ReadOnlyAccess {
					continue // these two specific policies are handled just above
				}
				if i := sort.SearchStrings(arns, arn); i == len(arns) || arns[i] != arn { // <https://pkg.go.dev/sort#Search>
					ui.Must(awsiam.DetachRolePolicy(ctx, accountCfg, *roleName, arn))
				}
			}
			if len(managedPolicyAttachments.ARNs) > 0 {
				ui.Spinf("attaching AWS-managed policies to the %s role in %s", *roleName, account)
				for _, arn := range managedPolicyAttachments.ARNs {
					ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, *roleName, arn))
				}
				ui.Stopf("ok")
			}

			if len(managedPolicyAttachments.Filenames) > 0 {
				ui.Spinf("merging custom policies for the %s role in %s", *roleName, account)
			} else {
				ui.Spinf("setting Substrate's minimal custom policy for the %s role in %s", *roleName, account)
			}
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
