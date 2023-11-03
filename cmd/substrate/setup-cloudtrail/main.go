package setupcloudtrail

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscloudtrail"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/humans"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	ManageCloudTrailFilename = "substrate.manage-cloudtrail"
	TrailName                = "GlobalMultiRegionOrganizationTrail"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate setup-cloudtrail")
		flag.PrintDefaults()
	}
	flag.Parse()

	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	substrateCfg := awscfg.Must(cfg.AssumeSubstrateRole(ctx, roles.Substrate, time.Hour))

	prefix := naming.Prefix()
	region := regions.Default()

	// Ensure the audit account exists.  This one comes first so we can enable
	// CloudTrail ASAP.  We might be _too_ fast, though, so we accommodate AWS
	// being a little slow in bootstrapping the organization for this the first
	// of several account creations.
	ui.Spin("finding or creating the audit account")
	auditAccount, err := mgmtCfg.FindSpecialAccount(ctx, accounts.Audit)
	ui.Must(err)
	if auditAccount == nil {
		ui.Stop("not found")
		reuse, err := ui.Confirm("does your AWS organization already have an account that stores audit logs which you'd like Substrate to use? (yes/no)")
		ui.Must(err)
		if reuse {
			auditAccountId, err := ui.Prompt("enter the account number of your existing audit account:")
			ui.Must(err)
			ui.Spin("adopting your existing audit account")
			ui.Must(awsorgs.Tag(ctx, mgmtCfg, auditAccountId, tagging.Map{
				// not tagging.Manager: tagging.Substrate because that's kind of a lie in this case
				tagging.SubstrateSpecialAccount: accounts.Audit,
				tagging.SubstrateType:           accounts.Audit,
			})) // this also ensures the account is in the organization
		} else {
			ui.Spin("creating the audit account")
		}
	}
	auditAccount, err = awsorgs.EnsureSpecialAccount(ctx, mgmtCfg, accounts.Audit)
	ui.Must(err)
	auditCfg, err := mgmtCfg.AssumeRole(ctx, aws.ToString(auditAccount.Id), roles.AuditAdministrator, time.Hour)
	if err != nil {
		auditCfg, err = mgmtCfg.AssumeRole(ctx, aws.ToString(auditAccount.Id), roles.OrganizationAccountAccessRole, time.Hour)
	}
	ui.Must(err)
	ui.Must(accounts.CheatSheet(ctx, mgmtCfg))
	ui.Stopf("account %s", auditAccount.Id)
	//log.Printf("%+v", auditAccount)

	// Ensure CloudTrail is permanently enabled organization-wide (unless
	// they opt-out).
	if !fileutil.Exists(ManageCloudTrailFilename) {
		ui.Spin("scoping out your organization's CloudTrail configuration(s)")
		trails, err := awscloudtrail.DescribeTrails(ctx, mgmtCfg)
		ui.Must(err)
		count := 0
		for _, trail := range trails {

			// If the Substrate-managed trail exists, presume that they opted
			// in or would have if they'd been given a choice by the earlier
			// version of Substrate that bootstrapped their management account.
			if aws.ToString(trail.Name) == TrailName { // TODO check more conditions? (IsMultiRegionTrail, IsOrganizationTrail, S3BucketName)
				ui.Must(os.WriteFile(ManageCloudTrailFilename, []byte("yes\n"), 0666))

			} else {
				count++
			}
		}
		if count > 0 {
			ui.Stopf("found %d extra trails", count)
			ui.Print("having more than one CloudTrail configuration in an AWS organization can be very expensive")
		} else {
			ui.Stop("ok")
		}

	}
	manageCloudTrail, err := ui.ConfirmFile(
		ManageCloudTrailFilename,
		`do you want Substrate to create and manage a CloudTrail configuration? (yes/no)`,
	)
	ui.Must(err)
	if manageCloudTrail {
		ui.Spin("configuring CloudTrail for your organization (every account, every region)")
		bucketName := fmt.Sprintf("%s-cloudtrail", prefix)
		ui.Must(awss3.EnsureBucket(
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
		))
		ui.Must(awsorgs.EnableAWSServiceAccess(ctx, mgmtCfg, "cloudtrail.amazonaws.com"))
		trail, err := awscloudtrail.EnsureTrail(ctx, mgmtCfg, TrailName, bucketName)
		ui.Must(err)
		ui.Stopf("bucket %s, trail %s", bucketName, trail.Name)
	}

	// Find or create the AuditAdministrator role.
	extraAdministrator, err := policies.ExtraAdministratorAssumeRolePolicy()
	ui.Must(err)
	auditRole, err := awsiam.EnsureRole(
		ctx,
		auditCfg,
		roles.AuditAdministrator,
		policies.Merge(
			policies.AssumeRolePolicyDocument(&policies.Principal{
				AWS: []string{
					roles.ARN(substrateCfg.MustAccountId(ctx), roles.Administrator),
					roles.ARN(auditCfg.MustAccountId(ctx), roles.AuditAdministrator), // allow this role to assume itself
					roles.ARN(mgmtCfg.MustAccountId(ctx), roles.Substrate),
					roles.ARN(substrateCfg.MustAccountId(ctx), roles.Substrate),
					users.ARN(mgmtCfg.MustAccountId(ctx), users.Substrate),
					users.ARN(substrateCfg.MustAccountId(ctx), users.Substrate),
				},
			}),
			extraAdministrator,
		),
	)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, auditCfg, auditRole.Name, policies.AdministratorAccess))

	// Find or create the Auditor role in the audit account. We do this even
	// if we didn't actually setup CloudTrail because having an audit account
	// with a Substrate-style Auditor role is still valuable.
	ui.Spin("configuring IAM in the audit account")
	auditorAssumeRolePolicy, err := humans.AuditorAssumeRolePolicy(ctx, mgmtCfg)
	ui.Must(err)
	auditorRole, err := awsiam.EnsureRole(ctx, auditCfg, roles.Auditor, auditorAssumeRolePolicy)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, auditCfg, auditorRole.Name, policies.ReadOnlyAccess))
	allowAssumeRole, err := awsiam.EnsurePolicy(ctx, auditCfg, policies.AllowAssumeRoleName, policies.AllowAssumeRole)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, auditCfg, auditorRole.Name, aws.ToString(allowAssumeRole.Arn)))
	denySensitiveReads, err := awsiam.EnsurePolicy(ctx, auditCfg, policies.DenySensitiveReadsName, policies.DenySensitiveReads)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, auditCfg, auditorRole.Name, aws.ToString(denySensitiveReads.Arn)))
	//log.Print(jsonutil.MustString(auditorRole))
	ui.Stop("ok")

}

// Synopsis returns a one-line, short synopsis of the command.
func Synopsis() string {
	return "configures CloudTrail for your organization"
}
