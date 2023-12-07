package accounts

import (
	"context"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/humans"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func SetupIAM(
	ctx context.Context,
	mgmtCfg, networkCfg, substrateCfg, accountCfg *awscfg.Config,
	domain, environment, quality string,
) {

	ui.Must(awsorgs.Tag(
		ctx,
		mgmtCfg,
		accountCfg.MustAccountId(ctx),
		tagging.Map{
			tagging.Domain:      domain,
			tagging.Environment: environment,
			tagging.Manager:     tagging.Substrate,
			//tagging.Name: awsorgs.NameFor(domain, environment, quality), // don't override this in case it was an invited account with an important name
			tagging.Quality:          quality,
			tagging.SubstrateVersion: version.Version,
		},
	))

	ui.Must(CheatSheet(ctx, awscfg.Must(mgmtCfg.OrganizationReader(ctx))))

	ui.Spin("configuring IAM")
	ui.Must2(humans.EnsureAdministratorRole(ctx, mgmtCfg, accountCfg))
	ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, accountCfg))

	// TODO if the legacy network account exists, ensure there's a network for this service account there
	// TODO if not, create (with confirmation) a network account for this environment and quality, peer it, and pass it along

	ui.Must2(terraform.EnsureStateManager(ctx, substrateCfg))

	// Create CloudWatch's cross-account sharing role in this account.
	//
	// This probably shouldn't be a core part of Substrate but it has been
	// for longer than Substrate had custom role management and would be
	// a bit troublesome to remove now.
	const cloudwatchRoleName = "CloudWatch-CrossAccountSharingRole"
	orgAssumeRolePolicy, err := awsorgs.OrgAssumeRolePolicy(ctx, mgmtCfg)
	ui.Must(err)
	cloudwatchRole, err := awsiam.EnsureRole(ctx, accountCfg, cloudwatchRoleName, orgAssumeRolePolicy)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/job-function/ViewOnlyAccess"))
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/AWSXrayReadOnlyAccess"))
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/CloudWatchReadOnlyAccess"))
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/CloudWatchAutomaticDashboardsAccess"))
	ui.Stop("ok")

	// TODO create Substrate-managed IAM roles that match this account.

}
