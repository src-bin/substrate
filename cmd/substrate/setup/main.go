package setup

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsram"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/humans"
	"github.com/src-bin/substrate/jsonutil"
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

var autoApprove, ignoreServiceQuotas, noApply *bool // shameful package variables to avoid rewriting bootstrap-{deploy,network}-account

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	autoApprove = flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	ignoreServiceQuotas = flag.Bool("ignore-service-quotas", false, "ignore service quotas appearing to be exhausted and continue anyway")
	noApply = flag.Bool("no-apply", false, "do not apply Terraform changes")
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate setup [-auto-approve] [-ignore-service-quotas] [-no-apply]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if version.IsTrial() {
		ui.Print("since this is a trial version of Substrate, it will post non-sensitive and non-personally identifying telemetry (documented in more detail at <https://docs.src-bin.com/substrate/ref/telemetry>) to Source & Binary to better understand how Substrate is being used; paying customers may opt out of this telemetry")
	} else {
		_, err := ui.ConfirmFile(
			telemetry.Filename,
			"can Substrate post non-sensitive and non-personally identifying telemetry (documented in more detail at <https://docs.src-bin.com/substrate/ref/telemetry>) to Source & Binary to better understand how Substrate is being used? (yes/no)",
		)
		ui.Must(err)
	}

	//log.Print(jsonutil.MustString(cfg.MustGetCallerIdentity(ctx)))
	if _, err := cfg.GetCallerIdentity(ctx); err != nil {
		_, err := cfg.SetRootCredentials(ctx)
		ui.Must(err)
	}
	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.Substrate, // triggers affordances for using (deprecated) OrganizationAdministrator role, too
		time.Hour,
	))
	//log.Print(jsonutil.MustString(mgmtCfg.MustGetCallerIdentity(ctx)))

	versionutil.PreventDowngrade(ctx, mgmtCfg)

	naming.Prefix()

	region := regions.Default()
	mgmtCfg = mgmtCfg.Regional(region)
	_, err := regions.Select()
	ui.Must(err)

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
		ui.Must(ioutil.WriteFile(naming.QualitiesFilename, []byte("default\n"), 0666))
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
			ui.Print("if this is not correct, press ^C and re-run this command with -fully-interactive")
			time.Sleep(5e9) // give them a chance to ^C
		}
	}
	ui.Must(veqpDoc.Validate(environments, qualities))
	//log.Printf("%+v", veqpDoc)

	// Ask them the expensive question about NAT Gateways.
	_, err = ui.ConfirmFile(
		networks.NATGatewaysFilename,
		`do you want to provision NAT Gateways for IPv4 traffic from your private subnets to the Internet? (yes/no; answering "yes" costs about $100 per month per region per environment/quality pair)`,
	)
	ui.Must(err)

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
					ui.Must(ioutil.WriteFile(EnforceIMDSv2Filename, []byte("yes\n"), 0666))
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
	mgmtAccountId := mgmtCfg.MustAccountId(ctx)
	ui.Must(awsorgs.Tag(ctx, mgmtCfg, mgmtAccountId, tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateType:    accounts.Management,
		tagging.SubstrateVersion: version.Version,
	}))

	// EnableAllFeatures, which is complicated but necessary in case an
	// organization was created as merely a consolidated billing organization.
	// This hasn't been a problem in three years so it doesn't seem worth the
	// effort until we encounter billing-only organizations in the real world
	// that are trying to adopt Substrate.

	// Ensure service control policies are enabled and that Substrate's is
	// attached and up-to-date. This is pretty basic and may eventually be
	// expanded into `substrate create-scp|scps` commands; there's also an
	// opportunity in managing tagging policies that requires more research.
	ui.Spin("configuring your organization's service control policy")
	ui.Must(awsorgs.EnablePolicyType(ctx, mgmtCfg, awsorgs.SERVICE_CONTROL_POLICY))
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

	// Find or create the Substrate account, upgrading an admin account if
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
		log.Print(jsonutil.MustString(adminAccounts))
		if i := len(adminAccounts); i > 1 {
			ui.Fatal("found more than one (deprecated) admin account")
		} else if i == 0 {
			ui.Fatal("(deprecated) admin account not found")
		}
		substrateAccount = adminAccounts[0]
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
	ui.Stopf("found %s", substrateAccount)

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

	// Find or create the Substrate role in the management account, possibly
	// without some of the principals that need to be able to assume this
	// role. It will be recreated later with all the princpals once they've
	// definitely been created.
	mgmtPrincipals := []string{
		roles.ARN(mgmtAccountId, roles.Substrate), // allow this role to assume itself
		aws.ToString(mgmtUser.Arn),
		aws.ToString(substrateUser.Arn),
	}
	if administratorRole, err := awsiam.GetRole(ctx, substrateCfg, roles.Administrator); err == nil {
		mgmtPrincipals = append(mgmtPrincipals, administratorRole.ARN)
	}
	if substrateRole, err := awsiam.GetRole(ctx, substrateCfg, roles.Substrate); err == nil {
		mgmtPrincipals = append(mgmtPrincipals, substrateRole.ARN)
	}
	mgmtRole, err := awsiam.EnsureRole(ctx, mgmtCfg, roles.Substrate, policies.AssumeRolePolicyDocument(&policies.Principal{
		AWS: mgmtPrincipals,
	}))
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, mgmtCfg, mgmtRole.Name, policies.AdministratorAccess))
	//log.Print(jsonutil.MustString(mgmtRole))

	// Find or create the Substrate role in the Substrate account. This is what
	// the Intranet will use. We'll try to allow the Substrate role in the
	// management account to assume this role but if it doesn't exist yet we'll
	// try again later.
	ui.Spin("configuring IAM in the Substrate account")
	substrateAssumeRolePolicy := policies.AssumeRolePolicyDocument(&policies.Principal{
		AWS: []string{
			mgmtRole.ARN,
			roles.ARN(substrateAccountId, roles.Substrate), // allow this role to assume itself
			aws.ToString(mgmtUser.Arn),
			aws.ToString(substrateUser.Arn),
		},
		Service: []string{"lambda.amazonaws.com"},
	})
	substrateRole, err := awsiam.EnsureRole(ctx, substrateCfg, roles.Substrate, substrateAssumeRolePolicy)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, substrateCfg, substrateRole.Name, policies.AdministratorAccess))
	ui.Must(awsiam.AttachRolePolicy(ctx, substrateCfg, substrateRole.Name, policies.AmazonAPIGatewayPushToCloudWatchLogs))
	//log.Print(jsonutil.MustString(substrateRole))

	// Find or create the Administrator and Auditor roles in the Substrate
	// account. These are the default roles to assign to humans in the IdP.
	administratorRole, err := humans.EnsureAdministratorRole(ctx, mgmtCfg, substrateCfg)
	ui.Must(err)
	ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, substrateCfg))
	ui.Stop("ok")

	// Update the Substrate role in the management account. We created it
	// earlier so we could reference it in IAM policies but it might not be
	// complete. This is how we'll eventually create accounts, etc.
	ui.Spin("configuring IAM in the management account")
	mgmtRole, err = awsiam.EnsureRole(
		ctx,
		mgmtCfg,
		roles.Substrate,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: []string{
			administratorRole.ARN,
			roles.ARN(mgmtAccountId, roles.Substrate), // allow this role to assume itself
			substrateRole.ARN,
			aws.ToString(mgmtUser.Arn),
			aws.ToString(substrateUser.Arn),
		}}),
	)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, mgmtCfg, mgmtRole.Name, policies.AdministratorAccess))
	//log.Print(jsonutil.MustString(mgmtRole))

	// Refresh our AWS SDK config for the management account because it might
	// be using the OrganizationAdministrator role. Now that we've created the
	// Substrate role in the management account, we can be sure this config
	// will actually use it and no longer have to worry about authorizing
	// OrganizationAdministrator to assume roles.
	for {
		mgmtCfg = awscfg.Must(cfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
		//log.Print(jsonutil.MustString(mgmtCfg.MustGetCallerIdentity(ctx)))
		if name, _ := roles.Name(aws.ToString(mgmtCfg.MustGetCallerIdentity(ctx).Arn)); name == roles.Substrate {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	ui.Stop("ok")

	// Find or create the {Deploy,Network,Organization}Administrator roles and
	// matching Auditor roles in all the special accounts that we can. The only
	// one that's a guarantee is the management account.
	ui.Spin("configuring additional administrative IAM roles")
	extraAdministrator, err := policies.ExtraAdministratorAssumeRolePolicy()
	ui.Must(err)
	deployCfg, err := mgmtCfg.AssumeSpecialRole(ctx, accounts.Deploy, roles.DeployAdministrator, time.Hour)
	if err != nil {
		deployCfg, err = mgmtCfg.AssumeSpecialRole(ctx, accounts.Deploy, roles.OrganizationAccountAccessRole, time.Hour)
	}
	if err == nil {
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
		ui.Print(" could not assume the DeployAdministrator role; continuing without managing its policies")
	}
	networkCfg, err := mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour)
	if err != nil {
		networkCfg, err = mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.OrganizationAccountAccessRole, time.Hour)
	}
	if err == nil {
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
		ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, networkCfg))
	} else {
		ui.Print(" could not assume the NetworkAdministrator role; continuing without managing its policies")
	}
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
	_, err = awsiam.EnsureRoleWithPolicy(
		ctx,
		mgmtCfg,
		roles.OrganizationReader,
		&policies.Document{
			Statement: []policies.Statement{{
				Principal: &policies.Principal{AWS: []string{"*"}},
				Action:    []string{"sts:AssumeRole"},
				Condition: policies.Condition{"StringEquals": {
					"aws:PrincipalOrgID": []string{aws.ToString(org.Id)},
				}},
			}},
		},
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

	// Ensure every account can run Terraform with remote state centralized
	// in the Substrate account. This is better than storing state in each
	// account because it minimizes the number of non-Terraform-managed
	// resources in all those other Terraform-using accounts.
	_, err = terraform.EnsureStateManager(ctx, substrateCfg)
	ui.Must(err)

	// Find or create Administrator and Auditor roles in every service account.
	ui.Spin("configuring Administrator and Auditor IAM roles in every service account")
	allAccounts, err := awsorgs.ListAccounts(ctx, cfg)
	ui.Must(err)
	for _, a := range allAccounts {
		if a.Tags[tagging.Domain] != "" && a.Tags[tagging.Domain] != naming.Admin || a.Tags[tagging.SubstrateType] == accounts.Service {
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
		}
	}
	ui.Stop("ok")

	// Generate, plan, and apply the legacy deploy account's Terraform code,
	// if the account exists.
	deploy(ctx, mgmtCfg)

	// Generate, plan, and apply the legacy network account's Terraform code,
	// if the account exists.
	network(ctx, mgmtCfg)

	// Configure the Intranet in the Substrate account.
	dnsDomainName, idpName := intranet(ctx, mgmtCfg)

	// TODO configure IAM Identity Center (later)

	// Render a "cheat sheet" of sorts that has all the account numbers, role
	// names, and role ARNs that folks might need to get the job done.
	ui.Must(accounts.CheatSheet(ctx, mgmtCfg))

	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
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
	ui.Print("")
	ui.Print("next steps:")
	ui.Print("- run `substrate create-account` to create service accounts to host your infrastructure")
	ui.Printf("- use `eval $(substrate credentials)` or <https://%s/credential-factory> to mint short-lived AWS access keys", dnsDomainName)
	switch idpName {
	case oauthoidc.AzureAD:
		ui.Print("- onboard your coworkers by setting the AWS.RoleName custom security attribute in Azure AD")
		ui.Print("  (see <https://docs.src-bin.com/substrate/bootstrapping/integrating-your-identity-provider/azure-ad> for details)")
	case oauthoidc.Google:
		ui.Print("- onboard your coworkers by setting the AWS.RoleName custom attribute in Google Workspace")
		ui.Print("  (see <https://docs.src-bin.com/substrate/bootstrapping/integrating-your-identity-provider/google> for details)")
	case oauthoidc.Okta:
		ui.Print("- onboard your coworkers by setting the AWS_RoleName profile attribute in Okta")
		ui.Print("  (see <https://docs.src-bin.com/substrate/bootstrapping/integrating-your-identity-provider/okta> for details)")
	}
	ui.Print("- refer to the Substrate documentation at <https://docs.src-bin.com/substrate/>")
	ui.Print("- email <help@src-bin.com> or mention us in Slack for support")

}
