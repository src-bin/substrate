package setup

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsram"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cmd/substrate/setup/cloudtrail"
	"github.com/src-bin/substrate/cmd/substrate/setup/debugger"
	deletestaticaccesskeys "github.com/src-bin/substrate/cmd/substrate/setup/delete-static-access-keys"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/features"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/humans"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

const (
	EnforceIMDSv2Filename    = "substrate.enforce-imdsv2"
	ServiceControlPolicyName = "SubstrateServiceControlPolicy"
)

var (
	runTerraform, autoApprove, noApply = new(bool), new(bool), new(bool)
	providersLock                      = new(bool)
	ignoreServiceQuotas                = new(bool)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup [--terraform [--auto-approve|--no-apply] [--providers-lock]] [--ignore-service-quotas]",
		Short: "setup Substrate in your AWS organization",
		Long: "`substrate setup`" + ` finds or creates your AWS organization, finds or creates the
AWS accounts and IAM principals Substrate uses to manage your organization, and
configures your Intranet to interact with your IdP; it is idempotent and safe
to run repeatedly`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--terraform", "--auto-approve", "--no-apply", "--providers-lock",
				"--ignore-service-quotas",
				"--fully-interactive", "--minimally-interactive", "--non-interactive",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().BoolVar(runTerraform, "terraform", false, "initialize and plan or apply Terraform in the special deploy and network accounts")
	cmd.Flags().BoolVar(autoApprove, "auto-approve", false, "with --terraform, apply Terraform changes without waiting for confirmation")
	cmd.Flags().BoolVar(noApply, "no-apply", false, "with --terraform, plan but do not apply Terraform changes")
	cmd.Flags().BoolVar(providersLock, "providers-lock", false, "with --terraform, run `terraform providers lock` during Terraform initialization")
	cmd.Flags().BoolVar(ignoreServiceQuotas, "ignore-service-quotas", false, "ignore the appearance of any service quota being exhausted and continue anyway")
	cmd.Flags().AddFlagSet(ui.InteractivityFlagSet())

	cmd.AddCommand(cloudtrail.Command())
	cmd.AddCommand(debugger.Command())
	cmd.AddCommand(deletestaticaccesskeys.Command())

	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {
	cmdutil.PrintRoot()

	//ui.Debug(cfg.MustGetCallerIdentity(ctx))
	regions.Default()
	ui.Must2(cfg.BootstrapCredentials(ctx)) // get from anywhere to IAM credentials so we can assume roles
	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.Substrate, // triggers affordances for using (deprecated) OrganizationAdministrator role, too
		time.Hour,
	))
	//ui.Debug(mgmtCfg.MustGetCallerIdentity(ctx))

	versionutil.PreventDowngrade(ctx, mgmtCfg)

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	naming.Prefix()
	ui.Must2(regions.Select())

	// Prompt for environments and qualities but make it less intimidating than
	// it was originally by leaving out the whole "admin" thing and by skipping
	// qualifies entirely, defaulting to "default", to avoid introducing that
	// advanced concept right out of the gate.
	environments, err := ui.EditFile(
		naming.EnvironmentsFilename,
		"the following environments are currently valid in your Substrate-managed infrastructure:",
		`list all your environments, one per line, in order of progression from e.g. "development" through e.g. "production"`,
	)
	ui.Must(err)
	for _, environment := range environments {
		if strings.ContainsAny(environment, " /") {
			ui.Fatal("environments cannot contain ' ' or '/'")
		}
		if environment == "peering" {
			ui.Fatal(`"peering" is a reserved environment name`)
		}
	}
	ui.Printf("using environments %s", strings.Join(environments, ", "))
	environments, err = naming.Environments()
	ui.Must(err)
	if !fileutil.Exists(naming.QualitiesFilename) {
		ui.Must(os.WriteFile(naming.QualitiesFilename, []byte("default\n"), 0666))
	}
	qualities, err := naming.Qualities()
	ui.Must(err)
	if len(qualities) == 0 {
		ui.Fatal("you must name at least one quality")
	}
	for _, quality := range qualities {
		if strings.ContainsAny(quality, " /") {
			ui.Fatal("qualities cannot contain ' ' or '/'")
		}
	}
	if len(qualities) > 1 {
		ui.Printf("using qualities %s", strings.Join(qualities, ", "))
	}

	// Combine all environments and qualities. If there's only one quality then
	// there's only one possible document; create it non-interactively. If
	// there's more than one quality, offer every combination that doesn't
	// appear in substrate.valid-environment-quality-pairs.json. Finally,
	// validate the document.
	veqpDoc, err := veqp.ReadDocument()
	ui.Must(err)
	if len(qualities) == 1 {
		for _, environment := range environments {
			veqpDoc.Ensure(environment, qualities[0])
		}
	} else {
		if len(veqpDoc.ValidEnvironmentQualityPairs) != 0 {
			ui.Print("you currently allow the following combinations of environment and quality in your Substrate-managed infrastructure:")
			for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
				ui.Printf("\t%-12s %s", eq.Environment, eq.Quality)
			}
		}
		if ui.Interactivity() == ui.FullyInteractive || ui.Interactivity() == ui.MinimallyInteractive && len(veqpDoc.ValidEnvironmentQualityPairs) == 0 {
			var ok bool
			if len(veqpDoc.ValidEnvironmentQualityPairs) != 0 {
				ok, err = ui.Confirm("is this correct? (yes/no)")
				ui.Must(err)
			}
			if !ok {
				for _, environment := range environments {
					for _, quality := range qualities {
						if !veqpDoc.Valid(environment, quality) {
							ok, err := ui.Confirmf(`do you want to allow %s-quality infrastructure in your %s environment? (yes/no)`, quality, environment)
							ui.Must(err)
							if ok {
								veqpDoc.Ensure(environment, quality)
							}
						}
					}
				}
			}
		} else {
			ui.Print("if this is not correct, press ^C and re-run this command with --fully-interactive")
			time.Sleep(5e9) // give them a chance to ^C
		}
	}
	ui.Must(veqpDoc.Validate(environments, qualities))
	//log.Printf("%+v", veqpDoc)

	// Ask them the expensive question about NAT Gateways.
	_, err = ui.ConfirmFile(
		networks.NATGatewaysFilename,
		`do you want to provision NAT Gateways for IPv4 traffic from your private subnets to the Internet? (yes/no; answering "yes" costs about $108 per month per region per environment)`,
	)
	ui.Must(err)

	// Ensure this account is (in) an organization.
	ui.Spin("finding or creating your organization")
	org, err := mgmtCfg.DescribeOrganization(ctx)
	if awsutil.ErrorCodeIs(err, awsorgs.AlreadyInOrganizationException) {

		// It seems impossible we'll hit this condition which has existed since
		// the initial commit but covers an error that doesn't obviously make
		// sense following DescribeOrganization and isn't documented as a
		// possible error from DescribeOrganization. The most likely
		// explanation is that lazy evaluation in the old awssessions package
		// resulted in an error here.
		ui.PrintWithCaller(err)

		err = nil // we presume this is the management account, to be proven later
	}
	if awsutil.ErrorCodeIs(err, awscfg.AWSOrganizationsNotInUseException) {

		// Create the organization since it doesn't yet exist.
		org, err = awsorgs.CreateOrganization(ctx, mgmtCfg)
		ui.Must(err)

	}
	ui.Must(err)
	orgAssumeRolePolicy, err := awsorgs.OrgAssumeRolePolicy(ctx, mgmtCfg)
	ui.Must(err)
	root, err := awsorgs.DescribeRoot(ctx, mgmtCfg)
	ui.Must(err)
	ui.Stopf("organization %s, root %s", org.Id, root.Id)
	//log.Printf("%+v", org)
	//log.Printf("%+v", root)

	// Ensure this is, indeed, the organization's management account.  This is
	// almost certainly redundant but I can't be bothered to read the reams
	// of documentation that it would take to prove this beyond a shadow of a
	// doubt so here we are wearing a belt and suspenders.
	ui.Spin("confirming the access key is from the organization's management account")
	callerIdentity := mgmtCfg.MustGetCallerIdentity(ctx)
	org = mgmtCfg.MustDescribeOrganization(ctx)
	if aws.ToString(callerIdentity.Account) != aws.ToString(org.MasterAccountId) {
		ui.Fatalf(
			"access key is from account %v instead of your organization's management account, %v",
			aws.ToString(callerIdentity.Account),
			aws.ToString(org.MasterAccountId),
		)
	}
	ui.Stop("ok")
	//log.Printf("%+v", callerIdentity)
	//log.Printf("%+v", org)

	// Tag the management account in the new style.
	ui.Spin("tagging the organization's management account")
	mgmtAccountId := mgmtCfg.MustAccountId(ctx)
	ui.Must(awsorgs.Tag(ctx, mgmtCfg, mgmtAccountId, tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateType:    accounts.Management,
		tagging.SubstrateVersion: version.Version,
	}))
	ui.Stop("ok")

	// EnableAllFeatures, which is complicated but necessary in case an
	// organization was created as merely a consolidated billing organization.
	// This hasn't been a problem in three years so it doesn't seem worth the
	// effort until we encounter billing-only organizations in the real world
	// that are trying to adopt Substrate.

	// Enable all the types of organization-wide policies.
	ui.Spin("enabling all AWS Organizations policy types")
	ui.Must(awsorgs.EnablePolicyType(ctx, mgmtCfg, awsorgs.AISERVICES_OPT_OUT_POLICY))
	ui.Must(awsorgs.EnablePolicyType(ctx, mgmtCfg, awsorgs.BACKUP_POLICY))
	ui.Must(awsorgs.EnablePolicyType(ctx, mgmtCfg, awsorgs.SERVICE_CONTROL_POLICY))
	ui.Must(awsorgs.EnablePolicyType(ctx, mgmtCfg, awsorgs.TAG_POLICY))
	ui.Stop("ok")

	// Maybe ask them about enforcing the use of IMDSv2. However, if their
	// existing service control policy requires that they use IMDSv2, don't
	// even offer the opportunity to allow the less secure configuration.
	// Note well, though, that because of this inference it's not sufficient
	// to delete the substrate.enforce-imdsv2 file in order to change this
	// configuration; to do that you also need to delete (or edit) the
	// service control policy.
	if !fileutil.Exists(EnforceIMDSv2Filename) {
		ui.Spin("scoping out your organization's service control policies")
		policySummaries, err := awsorgs.ListPolicies(ctx, cfg, awsorgs.SERVICE_CONTROL_POLICY)
		ui.Must(err)
		for _, policySummary := range policySummaries {
			if aws.ToString(policySummary.Name) == ServiceControlPolicyName {
				policy, err := awsorgs.DescribePolicy(ctx, cfg, aws.ToString(policySummary.Id))
				ui.Must(err)
				if strings.Contains(aws.ToString(policy.Content), `"ec2:RoleDelivery": "2.0"`) {
					ui.Must(os.WriteFile(EnforceIMDSv2Filename, []byte("yes\n"), 0666))
					break
				}
			}
		}
		ui.Stop("ok")
	}
	enforceIMDSv2, err := ui.ConfirmFile(
		EnforceIMDSv2Filename,
		`do you want to enforce the use of the EC2 IMDSv2 organization-wide? (yes/no; answering "yes" improves security posture but may break legacy EC2 workloads)`,
	)
	ui.Must(err)

	// Ensure service control policies are enabled and that Substrate's is
	// attached and up-to-date. This is pretty basic and may eventually be
	// expanded into `substrate scp create|delete|list` commands; there's also
	// an opportunity in managing tagging policies that requires more research.
	ui.Spin("configuring your organization's service control policy")
	statements := []policies.Statement{

		// Allow everything not explicitly denied. Bring it on.
		policies.Statement{
			Action:   []string{"*"},
			Resource: []string{"*"},
		},

		// It's catastrophically expensive to create a second trail so let's
		// not let anyone do it. Also don't let them delete the one existing
		// trail. It's really too bad this doesn't bind the managemente
		// account in the slightest since that's perhaps the most likely place
		// someone will try to create another trail.
		policies.Statement{
			Action: []string{
				"cloudtrail:CreateTrail",
				"cloudtrail:DeleteTrail",
			},
			Effect:   policies.Deny,
			Resource: []string{"*"},
		},
	}
	if enforceIMDSv2 {
		statements = append(
			statements,

			// Enforce exclusive IMDSv2 use at ec2:RunInstances.
			policies.Statement{
				Action: []string{"ec2:RunInstances"},
				Condition: policies.Condition{
					"StringNotEquals": {
						"ec2:MetadataHttpTokens": []string{"required"},
					},
				},
				Effect:   policies.Deny,
				Resource: []string{"arn:aws:ec2:*:*:instance/*"},
			},

			// Also enforce exclusive IMDSv2 use by voiding credentials from IMDSv1.
			policies.Statement{
				Action: []string{"*"},
				Condition: policies.Condition{
					"NumericLessThan": {
						"ec2:RoleDelivery": []string{"2.0"},
					},
				},
				Effect:   policies.Deny,
				Resource: []string{"*"},
			},
		)
	}
	ui.Must(awsorgs.EnsurePolicy(
		ctx,
		mgmtCfg,
		root,
		ServiceControlPolicyName,
		awsorgs.SERVICE_CONTROL_POLICY,
		&policies.Document{Statement: statements},
	))
	ui.Stop("ok")

	// Enable resource sharing throughout the organization.
	ui.Spin("enabling resource sharing throughout your organization")
	ui.Must(awsram.EnableSharingWithAwsOrganization(ctx, mgmtCfg))
	ui.Stop("ok")

	// Find or create the Substrate account, upgrading an old admin account if
	// that's all we can find. Tag it in the new style to close off the era of
	// `substrate bootstrap-*` and `substrate create-admin-account` for good.
	ui.Spin("finding the Substrate account")
	substrateAccount, err := mgmtCfg.FindSubstrateAccount(ctx)
	ui.Must(err)
	//log.Print(jsonutil.MustString(substrateAccount))
	if substrateAccount == nil { // maybe just haven't upgraded yet
		ui.Stop("not found")
		ui.Spin("finding an admin account to upgrade")
		adminAccounts, err := mgmtCfg.FindAdminAccounts(ctx)
		ui.Must(err)
		//log.Print(jsonutil.MustString(adminAccounts))
		if i := len(adminAccounts); i > 1 {
			ui.Fatal("found more than one (deprecated) admin account")
		} else if i == 1 {
			substrateAccount = adminAccounts[0]
		}
	}
	if substrateAccount == nil { // genuinely a new installation
		ui.Stop("not found")
		ui.Spin("creating the Substrate account")
		substrateAccount, err = awsorgs.EnsureSpecialAccount(ctx, cfg, accounts.Substrate)
		ui.Must(err)
	}
	substrateAccountId := aws.ToString(substrateAccount.Id)
	ui.Must(awsorgs.Tag(ctx, mgmtCfg, substrateAccountId, tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateType:    accounts.Substrate,
		tagging.SubstrateVersion: version.Version,
	}))
	substrateCfg, err := mgmtCfg.AssumeRole(ctx, substrateAccountId, roles.Substrate, time.Hour)
	if err != nil {
		substrateCfg, err = mgmtCfg.AssumeRole(ctx, substrateAccountId, roles.Administrator, time.Hour)
	}
	if err != nil {
		substrateCfg, err = mgmtCfg.AssumeRole(ctx, substrateAccountId, roles.OrganizationAccountAccessRole, time.Hour)
	}
	ui.Must(err)
	ui.Stop(substrateAccount)

	// Find or create the Substrate user in the management and Substrate
	// accounts. We need them both to exist early because we're about to
	// reference them both in some IAM policies.
	//
	// The one in the management account is there to accommodate switching from
	// root credentials to IAM credentials so that we can assume roles.
	//
	// The one in the Substrate account is how we'll mint 12-hour sessions all
	// over the organization.
	ui.Spin("finding or creating IAM users for bootstrapping and minting 12-hour credentials")
	mgmtUser, err := awsiam.EnsureUser(ctx, mgmtCfg, users.Substrate)
	ui.Must(err)
	ui.Must(awsiam.AttachUserPolicy(ctx, mgmtCfg, aws.ToString(mgmtUser.UserName), policies.AdministratorAccess))
	//log.Print(jsonutil.MustString(mgmtUser))
	substrateUser, err := awsiam.EnsureUser(ctx, substrateCfg, users.Substrate)
	ui.Must(err)
	ui.Must(awsiam.AttachUserPolicy(ctx, substrateCfg, aws.ToString(substrateUser.UserName), policies.AdministratorAccess))
	//log.Print(jsonutil.MustString(substrateUser))
	ui.Stop("ok")

	// Find or create the Substrate role in the management and Substrate
	// accounts, possibly without some principals that don't exist yet. Both
	// of these roles will be recreated later after all the principals have
	// definitely been created.
	ui.Spin("pre-creating IAM roles for circular references in assume-role policies")
	mgmtPrincipals := []string{
		roles.ARN(mgmtAccountId, roles.Substrate), // allow this role to assume itself
		aws.ToString(mgmtUser.Arn),
		aws.ToString(substrateUser.Arn),
	}
	substratePrincipals := []string{
		roles.ARN(substrateAccountId, roles.Substrate), // allow this role to assume itself
		aws.ToString(mgmtUser.Arn),
		aws.ToString(substrateUser.Arn),
	}
	if administratorRole, err := awsiam.GetRole(ctx, substrateCfg, roles.Administrator); err == nil {
		mgmtPrincipals = append(mgmtPrincipals, administratorRole.ARN)
		substratePrincipals = append(substratePrincipals, administratorRole.ARN)
	}
	if mgmtRole, err := awsiam.GetRole(ctx, mgmtCfg, roles.Substrate); err == nil {
		mgmtPrincipals = append(mgmtPrincipals, mgmtRole.ARN)
		substratePrincipals = append(substratePrincipals, mgmtRole.ARN)
	}
	if substrateRole, err := awsiam.GetRole(ctx, substrateCfg, roles.Substrate); err == nil {
		mgmtPrincipals = append(mgmtPrincipals, substrateRole.ARN)
		substratePrincipals = append(substratePrincipals, substrateRole.ARN)
	}
	mgmtRole, err := awsiam.EnsureRole(ctx, mgmtCfg, roles.Substrate, policies.AssumeRolePolicyDocument(&policies.Principal{
		AWS: mgmtPrincipals,
	}))
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, mgmtCfg, mgmtRole.Name, policies.AdministratorAccess))
	//log.Print(jsonutil.MustString(mgmtRole))
	substratePrincipals = append(substratePrincipals, mgmtRole.ARN)
	substrateRole, err := awsiam.EnsureRole(ctx, substrateCfg, roles.Substrate, policies.AssumeRolePolicyDocument(&policies.Principal{
		AWS:     substratePrincipals,
		Service: []string{"apigateway.amazonaws.com", "lambda.amazonaws.com"},
	}))
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, substrateCfg, substrateRole.Name, policies.AdministratorAccess))
	//log.Print(jsonutil.MustString(substrateRole))
	ui.Stop("ok")

	// Find or create the Administrator and Auditor roles in the Substrate
	// account. These are the default roles to assign to humans in the IdP.
	ui.Spin("finding or creating the Administrator and Auditor roles in the Substrate account")
	administratorRole, err := humans.EnsureAdministratorRole(ctx, mgmtCfg, substrateCfg)
	ui.Must(err)
	ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, substrateCfg))
	ui.Stop("ok")

	// Update the Substrate role in the management and Substrate accounts. We
	// created these earlier so we could reference them in IAM policies but
	// they might not be complete. These roles are what the Intranet and most
	// of the tools will use to create accounts, assume roles, and so on.
	ui.Spin("updating Substrate IAM roles in the management and Substrate accounts")
	mgmtRole, err = awsiam.EnsureRole(
		ctx,
		mgmtCfg,
		roles.Substrate,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: []string{
			administratorRole.ARN,
			mgmtRole.ARN, // allow this role to assume itself
			substrateRole.ARN,
			aws.ToString(mgmtUser.Arn),
			aws.ToString(substrateUser.Arn),
		}}),
	)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, mgmtCfg, mgmtRole.Name, policies.AdministratorAccess))
	//log.Print(jsonutil.MustString(mgmtRole))
	substrateAssumeRolePolicy := policies.AssumeRolePolicyDocument(&policies.Principal{
		AWS: []string{
			administratorRole.ARN,
			mgmtRole.ARN,
			substrateRole.ARN, // allow this role to assume itself
			aws.ToString(mgmtUser.Arn),
			aws.ToString(substrateUser.Arn),
		},
		Service: []string{"apigateway.amazonaws.com", "lambda.amazonaws.com"},
	})
	substrateRole, err = awsiam.EnsureRole(ctx, substrateCfg, roles.Substrate, substrateAssumeRolePolicy)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, substrateCfg, substrateRole.Name, policies.AdministratorAccess))
	ui.Must(awsiam.AttachRolePolicy(ctx, substrateCfg, substrateRole.Name, policies.AmazonAPIGatewayPushToCloudWatchLogs))
	//log.Print(jsonutil.MustString(substrateRole))
	ui.Stop("ok")

	// Refresh our AWS SDK config for the management account because it might
	// be using the OrganizationAdministrator role. Now that we've created the
	// Substrate role in the management account, we can be sure this config
	// will actually use it and no longer have to worry about authorizing
	// OrganizationAdministrator to assume roles.
	ui.Spin("refreshing AWS credentials for the management and Substrate accounts")
	for {
		if mgmtCfg, err = cfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour); err == nil {
			//log.Print(jsonutil.MustString(mgmtCfg.MustGetCallerIdentity(ctx)))
			if name, _ := roles.Name(aws.ToString(mgmtCfg.MustGetCallerIdentity(ctx).Arn)); name == roles.Substrate {
				break
			}
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	for {
		if substrateCfg, err = cfg.AssumeSubstrateRole(ctx, roles.Substrate, time.Hour); err == nil {
			//log.Print(jsonutil.MustString(substrateCfg.MustGetCallerIdentity(ctx)))
			if name, _ := roles.Name(aws.ToString(substrateCfg.MustGetCallerIdentity(ctx).Arn)); name == roles.Substrate {
				break
			}
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	ui.Stop("ok")

	// Create CloudWatch's service-linked role in the Substrate account and
	// its rather busted role in the management account for discovering
	// cross-account logs and metrics.
	//
	// This probably shouldn't be a core part of Substrate but it has been
	// for longer than Substrate had custom role management and would be
	// a bit troublesome to remove now.
	ui.Spin("creating service-linked role for CloudWatch")
	ui.Must2(awsiam.EnsureServiceLinkedRole(
		ctx,
		substrateCfg,
		"AWSServiceRoleForCloudWatchCrossAccount",
		"cloudwatch-crossaccount.amazonaws.com",
	))
	ui.Must2(awsiam.EnsureRoleWithPolicy(
		ctx,
		mgmtCfg,
		"CloudWatch-CrossAccountSharing-ListAccountsRole",
		orgAssumeRolePolicy,
		&policies.Document{
			Statement: []policies.Statement{{
				Action: []string{
					"organizations:ListAccounts",
					"organizations:ListAccountsForParent",
				},
				Resource: []string{"*"},
			}},
		},
	))
	ui.Stop("ok")

	// Find or create the {Audit,Deploy,Network,Organization}Administrator
	// roles and matching Auditor roles. Don't bother creating the audit or
	// deploy accounts if we can't find them. (The audit account may be
	// created later. The deploy account's former purposes are being taken on
	// by the Substrate account.) Create the network account if it doesn't
	// already exist. The management account must already exist because we're
	// in an organization.
	ui.Spin("configuring additional administrative IAM roles")
	extraAdministrator, err := policies.ExtraAdministratorAssumeRolePolicy()
	ui.Must(err)
	auditCfg, err := mgmtCfg.AssumeSpecialRole(ctx, accounts.Audit, roles.AuditAdministrator, time.Hour)
	if err != nil {
		auditCfg, err = mgmtCfg.AssumeSpecialRole(ctx, accounts.Audit, roles.OrganizationAccountAccessRole, time.Hour)
	}
	if err == nil {
		ui.Must2(awsorgs.EnsureSpecialAccount(ctx, mgmtCfg, accounts.Audit))
		ui.Must(humans.EnsureAuditAccountRoles(ctx, mgmtCfg, substrateCfg, auditCfg))
	} else {
		ui.Print(" could not assume the AuditAdministrator role; continuing without managing roles and policies in the audit account")
	}
	deployCfg, err := mgmtCfg.AssumeSpecialRole(ctx, accounts.Deploy, roles.DeployAdministrator, time.Hour)
	if err != nil {
		deployCfg, err = mgmtCfg.AssumeSpecialRole(ctx, accounts.Deploy, roles.OrganizationAccountAccessRole, time.Hour)
	}
	if err == nil {
		ui.Must2(awsorgs.EnsureSpecialAccount(ctx, mgmtCfg, accounts.Deploy))
		deployRole, err := awsiam.EnsureRole(
			ctx,
			deployCfg,
			roles.DeployAdministrator,
			policies.Merge(
				policies.AssumeRolePolicyDocument(&policies.Principal{
					AWS: []string{
						roles.ARN(substrateAccountId, roles.Administrator),
						roles.ARN(deployCfg.MustAccountId(ctx), roles.DeployAdministrator), // allow this role to assume itself
						mgmtRole.ARN,
						substrateRole.ARN,
						aws.ToString(mgmtUser.Arn),
						aws.ToString(substrateUser.Arn),
					},
				}),
				extraAdministrator,
			),
		)
		ui.Must(err)
		ui.Must(awsiam.AttachRolePolicy(ctx, deployCfg, deployRole.Name, policies.AdministratorAccess))
		//log.Print(jsonutil.MustString(deployRole))
		ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, deployCfg))
	} else {
		ui.Print(" could not assume the DeployAdministrator role; continuing without managing roles and policies in the (legacy) deploy account")
	}
	ui.Spinf("finding or creating the network account")
	networkAccount, err := awsorgs.EnsureSpecialAccount(ctx, mgmtCfg, accounts.Network)
	ui.Must(err)
	ui.Stop(networkAccount)
	networkCfg, err := mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour)
	if err != nil {
		networkCfg, err = mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.OrganizationAccountAccessRole, time.Hour)
	}
	if err != nil { // if tags are too eventually consistent
		networkCfg, err = mgmtCfg.AssumeRole(ctx, aws.ToString(networkAccount.Id), roles.OrganizationAccountAccessRole, time.Hour)
	}
	ui.Must(err)
	networkRole, err := awsiam.EnsureRole(
		ctx,
		networkCfg,
		roles.NetworkAdministrator,
		policies.Merge(
			policies.AssumeRolePolicyDocument(&policies.Principal{
				AWS: []string{
					roles.ARN(substrateAccountId, roles.Administrator),
					roles.ARN(networkCfg.MustAccountId(ctx), roles.NetworkAdministrator), // allow this role to assume itself
					mgmtRole.ARN,
					substrateRole.ARN,
					aws.ToString(mgmtUser.Arn),
					aws.ToString(substrateUser.Arn),
				},
			}),
			extraAdministrator,
		),
	)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, networkCfg, networkRole.Name, policies.AdministratorAccess))
	for {
		if networkCfg, err = mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour); err == nil {
			//log.Print(jsonutil.MustString(networkCfg.MustGetCallerIdentity(ctx)))
			if name, _ := roles.Name(aws.ToString(networkCfg.MustGetCallerIdentity(ctx).Arn)); name == roles.NetworkAdministrator {
				break
			}
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	ui.Stop("ok")
	ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, networkCfg))
	orgAdminRole, err := awsiam.EnsureRole(
		ctx,
		mgmtCfg,
		roles.OrganizationAdministrator,
		policies.Merge(
			policies.AssumeRolePolicyDocument(&policies.Principal{
				AWS: []string{
					roles.ARN(substrateAccountId, roles.Administrator),
					roles.ARN(mgmtAccountId, roles.OrganizationAdministrator), // allow this role to assume itself
					mgmtRole.ARN,
					substrateRole.ARN,
					aws.ToString(mgmtUser.Arn),
					aws.ToString(substrateUser.Arn),
				},
			}),
			extraAdministrator,
		),
	)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, mgmtCfg, orgAdminRole.Name, policies.AdministratorAccess))
	ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, mgmtCfg))
	ui.Stop("ok")

	// Find or create the legacy OrganizationReader role. Unlike the others,
	// we probably won't keep this one around long-term because it's not useful
	// as a general-purpose read-only role like Auditor is.
	//
	// We spin looking for an affirmative sign that we can assume this role
	// because AWS IAM is eventually consistent, we need to assume this role
	// immediately after this step, and we've lost this race before.
	_, err = awsiam.EnsureRoleWithPolicy(
		ctx,
		mgmtCfg,
		roles.OrganizationReader,
		orgAssumeRolePolicy,
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
	ui.Must(err)
	ui.Spin("testing the OrganizationReader role (because AWS IAM is eventually consistent)")
	for {
		if _, err := substrateCfg.OrganizationReader(ctx); err == nil { // same config as terraform.EnsureStateManager
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	ui.Stop("ok")

	// Ensure every account can run Terraform with remote state centralized
	// in the Substrate account. This is better than storing state in each
	// account because it minimizes the number of non-Terraform-managed
	// resources in all those other Terraform-using accounts.
	ui.Must2(terraform.EnsureStateManager(ctx, substrateCfg))
	ui.Spin("testing the TerraformStateManager role (because AWS IAM is eventually consistent)")
	for {
		if deployCfg == nil {
			_, err = networkCfg.AssumeSubstrateRole(ctx, roles.TerraformStateManager, time.Hour)
		} else {
			_, err = networkCfg.AssumeSpecialRole(ctx, accounts.Deploy, roles.TerraformStateManager, time.Hour)
		}
		if err == nil {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	ui.Stop("ok")

	// Find or create Administrator and Auditor roles in every service account.
	ui.Spin("configuring Administrator and Auditor IAM roles in every service account")
	allAccounts, err := awsorgs.ListAccounts(ctx, cfg)
	ui.Must(err)
	for _, a := range allAccounts {
		if a.Tags[tagging.Domain] != "" && a.Tags[tagging.Domain] != naming.Admin || a.Tags[tagging.SubstrateType] == accounts.Service {
			ui.Spin(a)
			//log.Print(jsonutil.MustString(a))
			ui.Must(awsorgs.Tag(ctx, mgmtCfg, aws.ToString(a.Id), tagging.Map{
				tagging.Manager:          tagging.Substrate,
				tagging.SubstrateType:    accounts.Service,
				tagging.SubstrateVersion: version.Version,
			}))
			cfg, err := a.Config(ctx, mgmtCfg, roles.Administrator, time.Hour)
			if err != nil {
				cfg, err = a.Config(ctx, mgmtCfg, roles.OrganizationAccountAccessRole, time.Hour)
			}
			ui.Must(err)
			ui.Must2(humans.EnsureAdministratorRole(ctx, mgmtCfg, cfg))
			ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, cfg))
			ui.Stop("ok")
		}
	}
	ui.Stop("ok")

	// Delegate organizational administration to the Substrate account. This
	// isn't strictly necessary for Substrate's operation but it appears that
	// AWS services are starting to assume this is in place for some of their
	// own cross-account functionality.
	//
	// We want to delegate organizations:* but, despite the UI suggesting that
	// would work, it doesn't. The list of APIs that the UI offers also doesn't
	// work. I've pared it back to the read-only APIs and that seems to work.
	// [1] suggests we should be able to delegate at least some of the write
	// APIs but that's not been our experience.
	//
	// [1] <https://stackoverflow.com/questions/75676727/aws-delegation-policy-error-this-resource-based-policy-contains-invalid-json>
	if features.DelegatedOrganizationAdministration.Enabled() {
		ui.Must(awsorgs.PutResourcePolicy(ctx, mgmtCfg, &policies.Document{
			Statement: []policies.Statement{{
				Action: []string{
					"organizations:Describe*",
					"organizations:List*",
				},

				// TODO Merge every principal from every statement in the
				// extraAdministrator policy into this principal so the customer's
				// principals can do all the delegated organization administration.
				// Without that, this is not actually stupendously useful to just
				// let the Substrate principals do things they're already adept at
				// doing by assuming a role in the management account.
				Principal: &policies.Principal{AWS: []string{
					substrateRole.ARN,
					aws.ToString(substrateUser.Arn),
				}},

				Resource: []string{"*"},
			}},
		}))
	}

	ui.Must(fileutil.WriteFileIfNotExists(
		terraform.RequiredVersionFilename,
		[]byte(fmt.Sprintln(terraform.RequiredVersion())),
	))
	ui.Must(fileutil.WriteFileIfNotExists(
		terraform.AWSProviderVersionConstraintFilename,
		[]byte(fmt.Sprintln(terraform.AWSProviderVersionConstraint())),
	))
	if *noApply {
		ui.Print("--no-apply given so not invoking `terraform apply`")
	}

	// Generate, plan, and apply the legacy deploy account's Terraform code,
	// if the account exists.
	deploy(ctx, mgmtCfg)

	// Configure the standard networks, one for the Substrate account and one
	// for each environment-quality pair, all in the network account and shared
	// with all the right service accounts. Generate functional Terraform
	// modules for each one that contains only data sources. Plan and/or apply
	// them because they still might contain resources that customers add.
	network(ctx, mgmtCfg)

	// Configure the Intranet in the Substrate account.
	dnsDomainName, idpName := intranet(ctx, mgmtCfg, substrateCfg)

	// If we find an IAM Identity Center installation, take it under our wing.
	if features.IdentityCenter.Enabled() {
		if !sso(ctx, mgmtCfg) {
			ui.Print("")
			ui.Print("no AWS IAM Identity Center configuration found")
			creds, err := mgmtCfg.Retrieve(ctx)
			ui.Must(err)
			ui.Print("if you want Substrate to manage AWS IAM Identity Center, follow these steps:")
			consoleSigninURL, err := federation.ConsoleSigninURL(
				creds,
				"https://console.aws.amazon.com/singlesignon/home", // destination
				nil,
			)
			ui.Must(err)
			ui.Printf("1. open the AWS Console in your management account <%s>", consoleSigninURL)
			ui.Print(`2. click "Enable" and follow the prompts to setup IAM Identity Center (because there's no API to do so)`)
			ui.Printf("3. repeat step 2 for all your regions (%s)", strings.Join(regions.Selected(), " "))
			ui.Print("4. re-run `substrate setup`")
		}
	}

	// Render a "cheat sheet" of sorts that has all the account numbers, role
	// names, and role ARNs that folks might need to get the job done.
	ui.Must(accounts.CheatSheet(ctx, mgmtCfg))

	if yesno, err := os.ReadFile(telemetry.Filename); err == nil {
		if strings.ToLower(fileutil.Tidy(yesno)) == "no" {
			ui.Print("")
			ui.Printf(`ignoring substrate.telemetry setting of "no"; telemetry is mandatory as of Substrate 2024.01; see <https://docs.substrate.tools/substrate/ref/telemetry>`)
		}
	}

	ui.Print("")
	ui.Print("setup complete!")
	ui.Print("next, let's get all the files Substrate has generated committed to version control")
	ui.Print("")
	ui.Print("ignore the following pattern in version control (i.e. add it to .gitignore):")
	ui.Print("")
	ui.Print(".substrate.*")
	ui.Print("")
	ui.Print("commit the following files and directories to version control:")
	ui.Print("")
	ui.Print("modules/")
	ui.Print("root-modules/")
	ui.Print("substrate.*")
	ui.Print(terraform.RequiredVersionFilename)
	ui.Print(terraform.AWSProviderVersionConstraintFilename)
	ui.Print("")
	ui.Print("next steps:")
	ui.Print("- run `substrate setup cloudtrail` to setup CloudTrail logging to S3 for all accounts in all regions")
	ui.Print("- run `substrate account create` to create service accounts to host and isolate your infrastructure")
	ui.Printf("- use `eval $(substrate credentials)` or <https://%s/credential-factory> to mint short-lived AWS access keys", dnsDomainName)
	switch idpName {
	case oauthoidc.AzureAD:
		ui.Print("- onboard your coworkers by setting the AWS.RoleName custom security attribute in Azure AD")
		ui.Print("  (see <https://docs.substrate.tools/substrate/bootstrapping/integrating-your-identity-provider/azure-ad> for details)")
	case oauthoidc.Google:
		ui.Print("- onboard your coworkers by setting the AWS.RoleName custom attribute in Google Workspace")
		ui.Print("  (see <https://docs.substrate.tools/substrate/bootstrapping/integrating-your-identity-provider/google> for details)")
	case oauthoidc.Okta:
		ui.Print("- onboard your coworkers by setting the AWS_RoleName profile attribute in Okta")
		ui.Print("  (see <https://docs.substrate.tools/substrate/bootstrapping/integrating-your-identity-provider/okta> for details)")
	}
	ui.Print("- refer to the Substrate documentation at <https://docs.substrate.tools/substrate/>")
	ui.Print("- email <help@src-bin.com> for support")

}
