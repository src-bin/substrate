package setup

import (
	"context"
	"flag"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate setup")
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

	if _, err := cfg.GetCallerIdentity(ctx); err != nil {
		if _, err := cfg.SetRootCredentials(ctx); err != nil {
			ui.Fatal(err)
		}
	}
	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.Substrate, // triggers affordances for using (deprecated) OrganizationAdministrator role, too
		time.Hour,       // XXX longer would be better since bootstrapping's expected to take some serious time
	))

	versionutil.PreventDowngrade(ctx, mgmtCfg)

	prefix := naming.Prefix()

	region := regions.Default()
	mgmtCfg = mgmtCfg.Regional(region)

	_ = prefix

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
	ui.Stopf("organization %s", org.Id)
	//log.Printf("%+v", org)

	// TODO EnableAllFeatures, which is complicated but necessary in case an
	// organization was created as merely a consolidated billing organization.

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
	ui.Must(awsorgs.Tag(ctx, mgmtCfg, mgmtCfg.MustAccountId(ctx), tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateType:    accounts.Management,
		tagging.SubstrateVersion: version.Version,
	}))

	// TODO Service Control Policy (or perhaps punt to a whole new `substrate create-scp|scps` family of commands; also tagging policies)

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
		// TODO create the Substrate account
	}
	ui.Must(awsorgs.Tag(ctx, mgmtCfg, aws.ToString(substrateAccount.Id), tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateType:    accounts.Substrate,
		tagging.SubstrateVersion: version.Version,
	}))
	substrateCfg := awscfg.Must(mgmtCfg.AssumeRole(
		ctx,
		aws.ToString(substrateAccount.Id),
		roles.OrganizationAccountAccessRole, // TODO try Administrator and Substrate, too, just in case this one's been deleted
		time.Hour,
	))
	ui.Stopf("found %s", substrateAccount)

	ui.Spin("finding or creating the Substrate IAM user and roles")

	// Find or create the Substrate role in the Substrate account. This is what
	// the Intranet will eventually use.
	substrateRole, err := awsiam.EnsureRole(
		ctx,
		substrateCfg,
		roles.Substrate,
		policies.AssumeRolePolicyDocument(&policies.Principal{
			Service: []string{"lambda.amazonaws.com"},
		}),
	)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(
		ctx,
		substrateCfg,
		substrateRole.Name,
		"arn:aws:iam::aws:policy/AdministratorAccess",
	))
	//log.Print(jsonutil.MustString(substrateRole))

	// Find or create the Substrate user in the Substrate account. This is how
	// we'll mint 12-hour sessions all over the organization.
	substrateUser, err := awsiam.EnsureUser(ctx, substrateCfg, users.Substrate)
	ui.Must(err)
	ui.Must(awsiam.AttachUserPolicy(
		ctx,
		substrateCfg,
		aws.ToString(substrateUser.UserName),
		"arn:aws:iam::aws:policy/AdministratorAccess",
	))
	//log.Print(jsonutil.MustString(substrateUser))

	// Find or create the Substrate role in the management account. This is
	// how we'll eventually create accounts, etc.
	mgmtRole, err := awsiam.EnsureRole(
		ctx,
		mgmtCfg,
		roles.Substrate,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: []string{
			substrateRole.ARN,
			aws.ToString(substrateUser.Arn),
		}}),
	)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(
		ctx,
		mgmtCfg,
		mgmtRole.Name,
		"arn:aws:iam::aws:policy/AdministratorAccess",
	))
	//log.Print(jsonutil.MustString(mgmtRole))

	ui.Stop("ok")

	// TODO create the TerraformStateManager role in the Substrate account

	// TODO ??? create legacy {Organization,Deploy,Network}Administrator and OrganizationReader roles ???

	// TODO create Administrator and Auditor roles in the Substrate account and every service account

	// TODO run the legacy deploy account's Terraform code, if the account exists

	// TODO run the legacy network account's Terraform code, if the account exists

	// TODO configure the Intranet

	// TODO configure IAM Identity Center (later)

	// TODO instructions on using the Credential Factory, Intranet, etc.

}
