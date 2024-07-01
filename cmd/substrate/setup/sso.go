package setup

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssso"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func sso(ctx context.Context, mgmtCfg *awscfg.Config) {

	// We actually can sort out this prerequisite for dealing with IAM Identity
	// Center. We'll do it now, before we look for instances and before we
	// prompt folks to go create their Identity Center instance. It seems like
	// we should need to enable this as ssoadmin.amazonaws.com since that's
	// where most of the APIs are but, empirically, it seems not.
	ui.Must(awsorgs.EnableAWSServiceAccess(ctx, mgmtCfg, "sso.amazonaws.com"))
	//ui.Must(awsorgs.EnableAWSServiceAccess(ctx, mgmtCfg, "ssoadmin.amazonaws.com"))

	instances, err := awssso.ListInstances(ctx, mgmtCfg)
	ui.Must(err)
	if len(instances) == 0 {
		ui.Print("")
		ui.Print("no AWS IAM Identity Center configuration found")
		creds, err := mgmtCfg.Retrieve(ctx)
		ui.Must(err)
		consoleSigninURL, err := federation.ConsoleSigninURL(
			creds,
			fmt.Sprintf("https://%s.console.aws.amazon.com/singlesignon/home", regions.Default()), // destination
			nil,
		)
		ui.Must(err)
		ui.Print("if you want Substrate to manage AWS IAM Identity Center, follow these steps:")
		ui.Printf("1. open the AWS Console in your management account <%s>", consoleSigninURL)
		ui.Print(`2. click "Enable" and follow the prompts to setup IAM Identity Center (because there's no API to do so)`)
		ui.Print("3. re-run `substrate setup`")
		ui.Print("")
		return
	}
	if len(instances) > 1 {
		ui.Fatalf("found %d instances of IAM Identity Center; more than one is supposed to be impossible", len(instances))
	}
	instance := instances[0]
	//ui.Debug(instance)

	// We've got an instance of Identity Center. Now let's figure out if we're
	// managing it with Substrate or leaving it alone. If we can't figure it
	// out we ask and record the answer (obliquely) in the instance's tags.
	// TODO It should be easier to walk back a "no" here than going into the
	// TODO AWS Console to manually change the Manager tag's value.
	if instance.Tags[tagging.Manager] == "" {
		if ui.Must2(ui.Confirm("should Substrate manage AWS IAM Identity Center? (yes/no)")) {
			ui.Must(awssso.TagInstance(ctx, mgmtCfg, instance, tagging.Map{
				tagging.Manager:          tagging.Substrate,
				tagging.SubstrateVersion: version.Version,
			}))
		} else {
			ui.Must(awssso.TagInstance(ctx, mgmtCfg, instance, tagging.Map{
				tagging.Manager: "NotSubstrate",
			}))
			return
		}
	} else if instance.Tags[tagging.Manager] != tagging.Substrate {
		ui.Printf("found IAM Identity Center instance %s but it's tagged Manager=NotSubstrate; not managing it", instance.InstanceArn)
		return
	}
	ui.Spinf("managing IAM Identity Center instance %s", instance.InstanceArn)

	ui.Spinf("finding or creating Administrators and Auditors groups in identity store %s", instance.IdentityStoreId)
	administrators := ui.Must2(awssso.EnsureGroup(ctx, mgmtCfg, instance, "Administrators"))
	auditors := ui.Must2(awssso.EnsureGroup(ctx, mgmtCfg, instance, "Auditors"))
	ui.Stop("ok")

	ui.Spin("finding or creating Administrator and Auditor permission sets")
	administrator := ui.Must2(awssso.EnsurePermissionSet(
		ctx,
		mgmtCfg,
		instance,
		roles.Administrator,
		[]string{policies.AdministratorAccess},
		nil,
		nil,
	))
	auditor := ui.Must2(awssso.EnsurePermissionSet(
		ctx,
		mgmtCfg,
		instance,
		roles.Auditor,
		[]string{policies.ReadOnlyAccess},
		[]string{policies.DenySensitiveReadsName},
		nil,
	))
	ui.Stop("ok")

	ui.Spin("provisioning Administrator and Auditor permission sets into all AWS accounts")
	for _, account := range ui.Must2(mgmtCfg.ListAccounts(ctx)) {
		ui.Must(awssso.EnsureGroupAccountAssignment(ctx, mgmtCfg, instance, administrator, aws.ToString(account.Id), administrators.GroupId))
		ui.Must(awssso.EnsureGroupAccountAssignment(ctx, mgmtCfg, instance, auditor, aws.ToString(account.Id), auditors.GroupId))
	}
	ui.Must(awssso.ProvisionPermissionSet(ctx, mgmtCfg, instance, administrator))
	ui.Must(awssso.ProvisionPermissionSet(ctx, mgmtCfg, instance, auditor))
	ui.Stop("ok")

	substrateAccount, err := mgmtCfg.FindSubstrateAccount(ctx)
	ui.Must(err)
	ui.Must(awsorgs.RegisterDelegatedAdministrator(
		ctx,
		mgmtCfg,
		aws.ToString(substrateAccount.Id),
		"sso.amazonaws.com",
	))

	ui.Stop("ok")
}
