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
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/networks"
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

	// Finally, ask them the expensive question about NAT Gateways.
	_, err = ui.ConfirmFile(
		networks.NATGatewaysFilename,
		`do you want to provision NAT Gateways for IPv4 traffic from your private subnets to the Internet? (yes/no; answering "yes" costs about $100 per month per region per environment/quality pair)`,
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
		substrateAccount, err = awsorgs.EnsureSpecialAccount(ctx, cfg, accounts.Substrate)
		ui.Must(err)
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

	// Ensure every account can run Terraform with remote state centralized
	// in the Substrate account. This is better than storing state in each
	// account because it minimizes the number of non-Terraform-managed
	// resources in all those other Terraform-using accounts.
	_, err = terraform.EnsureStateManager(ctx, substrateCfg)
	if awsutil.ErrorCodeIs(err, awss3.BucketAlreadyExists) { // take this as a sign that the bucket's in their (legacy) deploy account
		err = nil
	}
	ui.Must(err)

	// TODO ??? create legacy {Organization,Deploy,Network}Administrator and OrganizationReader roles ???

	// TODO create Administrator and Auditor roles in the Substrate account and every service account

	// TODO run the legacy deploy account's Terraform code, if the account exists
	deploy(ctx, mgmtCfg)

	// TODO run the legacy network account's Terraform code, if the account exists
	network(ctx, mgmtCfg)

	// TODO configure the Intranet
	dnsDomainName := intranet(ctx, substrateCfg)

	// TODO configure IAM Identity Center (later)

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
	ui.Print("then, run `substrate create-account` as you see fit to create the service accounts you need")
	ui.Printf("you should also start using `eval $(substrate credentials)` or <https://%s/credential-factory> to mint short-lived AWS access keys", dnsDomainName)
}
