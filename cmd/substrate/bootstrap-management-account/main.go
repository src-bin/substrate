package bootstrapmanagementaccount

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscloudtrail"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsram"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

const (
	EnforceIMDSv2Filename    = "substrate.enforce-imdsv2"
	ServiceControlPolicyName = "SubstrateServiceControlPolicy"
	TagPolicyName            = "SubstrateTaggingPolicy"
	TrailName                = "GlobalMultiRegionOrganizationTrail"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	flag.Usage = func() {
		ui.Print("Usage: substrate bootstrap-management-account")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()

	prefix := naming.Prefix()
	region := regions.Default()

	var err error
	if _, err = cfg.GetCallerIdentity(ctx); err != nil {
		if _, err = cfg.SetRootCredentials(ctx); err != nil {
			ui.Fatal(err)
		}
	}
	cfg = awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.OrganizationAdministrator,
		time.Hour,
	)).Regional(region)
	versionutil.PreventDowngrade(ctx, cfg)

	// Ensure this account is (in) an organization.
	ui.Spin("finding or creating your organization")
	org, err := cfg.DescribeOrganization(ctx)
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
		org, err = awsorgs.CreateOrganization(ctx, cfg)
		if err != nil {
			ui.Fatal(err)
		}

	}
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("organization %s", org.Id)
	//log.Printf("%+v", org)

	// TODO EnableAllFeatures, which is complicated but necessary in case an
	// organization was created as merely a consolidated billing organization.

	// Ensure this is, indeed, the organization's management account.  This is
	// almost certainly redundant but I can't be bothered to read the reams
	// of documentation that it would take to prove this beyond a shadow of a
	// doubt so here we are wearing a belt and suspenders.
	ui.Spin("confirming the access key is from the organization's management account")
	callerIdentity := cfg.MustGetCallerIdentity(ctx)
	org = cfg.MustDescribeOrganization(ctx)
	if aws.ToString(callerIdentity.Account) != aws.ToString(org.MasterAccountId) {
		log.Fatalf(
			"access key is from account %v instead of your organization's management account, %v",
			aws.ToString(callerIdentity.Account),
			aws.ToString(org.MasterAccountId),
		)
	}
	ui.Stop("ok")
	//log.Printf("%+v", callerIdentity)
	//log.Printf("%+v", org)

	cfg.Telemetry().FinalAccountId = aws.ToString(callerIdentity.Account)
	cfg.Telemetry().FinalRoleName = roles.OrganizationAdministrator

	// Ensure the audit account exists.  This one comes first so we can enable
	// CloudTrail ASAP.  We might be _too_ fast, though, so we accommodate AWS
	// being a little slow in bootstrapping the organization for this the first
	// of several account creations.
	ui.Spin("finding or creating the audit account")
	auditAccount, err := cfg.FindSpecialAccount(ctx, accounts.Audit)
	ui.Must(err)
	if auditAccount == nil {
		ui.Stop("not found")
		reuse, err := ui.Confirm("does your AWS organization already have an account that stores audit logs which you'd like Substrate to use? (yes/no)")
		ui.Must(err)
		if reuse {
			auditAccountId, err := ui.Prompt("enter the account number of your existing audit account:")
			ui.Must(err)
			ui.Spin("adopting your existing audit account")
			ui.Must(awsorgs.Tag(ctx, cfg, auditAccountId, tagging.Map{
				tagging.SubstrateSpecialAccount: accounts.Audit,
			})) // this also ensures the account is in the organization
		} else {
			ui.Spin("creating the audit account")
		}
	}
	auditAccount, err = awsorgs.EnsureSpecialAccount(ctx, cfg, accounts.Audit)
	ui.Must(err)
	ui.Stopf("account %s", auditAccount.Id)
	//log.Printf("%+v", auditAccount)

	// Ensure CloudTrail is permanently enabled organization-wide.
	ui.Spin("configuring CloudTrail for your organization (every account, every region)")
	auditCfg := awscfg.Must(cfg.AssumeRole(
		ctx,
		aws.ToString(auditAccount.Id),
		roles.OrganizationAccountAccessRole,
		time.Hour,
	))
	bucketName := fmt.Sprintf("%s-cloudtrail", prefix)
	if err := awss3.EnsureBucket(
		ctx,
		auditCfg,
		bucketName,
		region,
		&policies.Document{
			Statement: []policies.Statement{
				{
					Principal: &policies.Principal{AWS: []string{aws.ToString(auditAccount.Id)}},
					Action:    []string{"s3:*"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
					},
				},
				{
					Principal: &policies.Principal{Service: []string{"cloudtrail.amazonaws.com"}},
					Action:    []string{"s3:GetBucketAcl", "s3:PutObject"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/AWSLogs/*", bucketName),
					},
				},
			},
		},
	); err != nil {
		ui.Fatal(err)
	}
	if err := awsorgs.EnableAWSServiceAccess(ctx, cfg, "cloudtrail.amazonaws.com"); err != nil {
		ui.Fatal(err)
	}
	trail, err := awscloudtrail.EnsureTrail(ctx, cfg, TrailName, bucketName)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("bucket %s, trail %s", bucketName, trail.Name)

	// TODO THIS IS VERY DUBIOUSLY VALUABLE, PROBABLY DON'T DO IT
	// Ensure AWS Config is enabled in all the special accounts in every
	// region that's in use. Setup an aggregator, too, that can access all
	// the Config buckets in the organization.
	// - <https://docs.aws.amazon.com/config/latest/developerguide/gs-cli-subscribe.html>
	// - <https://docs.aws.amazon.com/config/latest/developerguide/set-up-aggregator-cli.html>
	// TODO IAM role with "arn:aws:iam::aws:policy/service-role/AWSConfigRoleForOrganizations" attached
	// TODO regions.Select()
	// TODO S3 buckets
	// TODO PutConfigurationRecorder
	// TODO PutDeliveryChannel
	// TODO StartConfigurationRecorder
	// TODO possibly another IAM role for aggregation
	// TODO delegated administrator for aggregation (the audit account)
	// TODO PutConfigurationAggregator, etc.
	// TODO might need to <https://docs.aws.amazon.com/config/latest/developerguide/authorize-aggregator-account-cli.html> for every account in `substrate create-account`

	// TODO THIS IS PROBABLY MUCH MORE DEFENSIBLY VALUABLE
	// Ensure AWS GuardDuty has delegated administration to the audit account,
	// is enabled in all existing accounts, and is tracking the organization
	// to enable itself in new accounts.
	// TODO EnableOrganizationAdminAccount
	// TODO regions.Select()
	// TODO CreateDetector
	// TODO CreateMembers (seems like I might be missing something about enabling GuardDuty for the management and audit accounts)
	// TODO InviteMembers with disableEmailNotification: true
	// TODO UpdateOrganizationConfiguration with autoEnable: true

	// Ensure the deploy and network accounts exist.
	ui.Spinf("finding or creating the deploy account")
	deployAccount, err := awsorgs.EnsureSpecialAccount(ctx, cfg, accounts.Deploy)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("account %s", deployAccount.Id)
	//log.Printf("%+v", deployAccount)
	ui.Spinf("finding or creating the network account")
	networkAccount, err := awsorgs.EnsureSpecialAccount(ctx, cfg, accounts.Network)
	if err != nil {
		ui.Fatal(err)
	}
	ui.Stopf("account %s", networkAccount.Id)
	//log.Printf("%+v", networkAccount)

	// Tag the management account.
	ui.Spin("tagging the management account")
	ui.Must(awsorgs.Tag(ctx, cfg, aws.ToString(org.MasterAccountId), map[string]string{
		tagging.Manager:                 tagging.Substrate,
		tagging.SubstrateSpecialAccount: accounts.Management,
		tagging.SubstrateVersion:        version.Version,
	}))
	ui.Stop("ok")

	// Render a "cheat sheet" of sorts that has all the account numbers, role
	// names, and role ARNs that folks might need to get the job done.
	ui.Must(accounts.CheatSheet(ctx, cfg))

	//ui.Spin("configuring your organization's service control and tagging policies")
	ui.Spin("configuring your organization's service control policy")

	// The management account isn't the organization, though.  It's just an account.
	// To affect the entire organization, we need its root.
	root, err := awsorgs.DescribeRoot(ctx, cfg)
	ui.Must(err)

	// Ensure service control policies are enabled and that Substrate's is
	// attached and up-to-date. As of 2023.02 we ask during the first run
	// (specifically, we ask whenever the policy doesn't already exist) if
	// we should enforce IMDSv2 via SCP. If this isn't the first run, we
	// presume IMDSv2 should be enforced as this is what we did in 2023.01
	// and prior versions.
	//
	// This MUST happen AFTER configuring CloudTrail.
	ui.Must(awsorgs.EnablePolicyType(ctx, cfg, awsorgs.SERVICE_CONTROL_POLICY))
	policySummaries, err := awsorgs.ListPolicies(ctx, cfg, awsorgs.SERVICE_CONTROL_POLICY)
	ui.Must(err)
	found := false
	for _, policySummary := range policySummaries {
		found = found || aws.ToString(policySummary.Name) == ServiceControlPolicyName
	}
	enforceIMDSv2 := true
	if found {
		err = ioutil.WriteFile(EnforceIMDSv2Filename, []byte("yes\n"), 0666)
	} else {
		enforceIMDSv2, err = ui.ConfirmFile(
			EnforceIMDSv2Filename,
			"do you want to enforce the use of the EC2 IMDSv2 organization-wide? (yes/no; improves security posture but may break legacy EC2 workloads)",
		)
	}
	ui.Must(err)
	statements := []policies.Statement{

		// Allow everything not explicitly denied. Bring it on.
		policies.Statement{
			Action:   []string{"*"},
			Resource: []string{"*"},
		},

		// It's catastrophically expensive to create a second trail
		// so let's not let anyone do it. Also don't let them delete
		// the one existing trail.
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
						"ec2:MetadataHttpTokens": "required",
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
						"ec2:RoleDelivery": "2.0",
					},
				},
				Effect:   policies.Deny,
				Resource: []string{"*"},
			},
		)
	}
	ui.Must(awsorgs.EnsurePolicy(
		ctx,
		cfg,
		root,
		ServiceControlPolicyName,
		awsorgs.SERVICE_CONTROL_POLICY,
		&policies.Document{Statement: statements},
	))
	/*
		policySummaries, err = awsorgs.ListPolicies(ctx, cfg, awsorgs.SERVICE_CONTROL_POLICY)
		ui.Must(err)
		for _, policySummary := range policySummaries {
			log.Print(jsonutil.MustString(policySummary))
		}
		//*/

	// Ensure tagging policies are enabled and that Substrate's is attached
	// and up-to-date.
	/*
		if err := awsorgs.EnsurePolicy(
			ctx,
			cfg,
			root,
			TagPolicyName,
			awsorgs.TAG_POLICY,
			`{"tags":{}}`,
		); err != nil {
			log.Fatal(err)
		}
	*/
	/*
		for policySummary := range awsorgs.ListPolicies(ctx, cfg, awsorgs.TAG_POLICY) {
			log.Printf("%+v", policySummary)
		}
		//*/

	ui.Stop("ok")

	// Enable resource sharing throughout the organization.
	ui.Spin("enabling resource sharing throughout your organization")
	ui.Must(awsram.EnableSharingWithAwsOrganization(ctx, cfg))
	ui.Stop("ok")

	admin.EnsureAdminRolesAndPolicies(ctx, cfg, true) // could detect if we created any special accounts but this way there's a simple do-it-anyway option if things get out of sync

	ui.Print("next, commit the following files to version control:")
	ui.Print("")
	ui.Print("substrate.*")
	ui.Print("")
	ui.Print("then, ignore the following pattern in version control (i.e. add it to .gitignore):")
	ui.Print("")
	ui.Print(".substrate.*")
	ui.Print("")
	ui.Print("then, run `substrate bootstrap-network-account`")

	// At the very, very end, when we're exceedingly confident in the
	// capabilities of the other accounts, detach the FullAWSAccess policy
	// from the management account.
	//
	// It's not clear to me that this is EVER a state we'll reach.  It's very
	// tough to give away one's ultimate get-out-of-jail-free card, after all.
	//
	// A safer step would be to attach a policy that allowed re-attaching the
	// FullAWSAccess policy before detaching it.  That would prevent accidental
	// use of the management account without being a "one-way door."

}
