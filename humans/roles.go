package humans

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

func EnsureAdministratorRole(ctx context.Context, mgmtOrSubstrateCfg, cfg *awscfg.Config) (*awsiam.Role, error) {
	assumeRolePolicy, err := AdministratorAssumeRolePolicy(ctx, mgmtOrSubstrateCfg)
	if err != nil {
		return nil, err
	}

	role, err := awsiam.EnsureRole(ctx, cfg, roles.Administrator, assumeRolePolicy)
	if err != nil {
		return nil, err
	}

	if err := awsiam.AttachRolePolicy(ctx, cfg, role.Name, policies.AdministratorAccess); err != nil {
		return nil, err
	}

	//log.Print(jsonutil.MustString(role))
	return role, nil
}

func EnsureAuditorRole(ctx context.Context, mgmtOrSubstrateCfg, cfg *awscfg.Config) (*awsiam.Role, error) {
	assumeRolePolicy, err := AuditorAssumeRolePolicy(ctx, mgmtOrSubstrateCfg)
	if err != nil {
		return nil, err
	}

	role, err := awsiam.EnsureRole(ctx, cfg, roles.Auditor, assumeRolePolicy)
	if err != nil {
		return nil, err
	}

	if err := awsiam.AttachRolePolicy(ctx, cfg, role.Name, policies.ReadOnlyAccess); err != nil {
		return nil, err
	}
	allowAssumeRole, err := awsiam.EnsurePolicy(ctx, cfg, policies.AllowAssumeRoleName, policies.AllowAssumeRole)
	if err != nil {
		return nil, err
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, role.Name, aws.ToString(allowAssumeRole.Arn)); err != nil {
		return nil, err
	}
	denySensitiveReads, err := awsiam.EnsurePolicy(ctx, cfg, policies.DenySensitiveReadsName, policies.DenySensitiveReads)
	if err != nil {
		return nil, err
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, role.Name, aws.ToString(denySensitiveReads.Arn)); err != nil {
		return nil, err
	}

	//log.Print(jsonutil.MustString(role))
	return role, nil
}

// EnsureAuditAccountRoles finds or creates the AuditAdministrator and
// Auditor roles in the audit account. These roles are specific to the audit
// account. The AuditAdministrator name obviously follows the convention for
// special accounts but additionally the Auditor role in the audit account
// _is_ allowed s3:GetObject and the other APIs typically denied to Auditor
// roles by the SubstrateDenySensitiveReads policy, which is not attached.
func EnsureAuditAccountRoles(ctx context.Context, mgmtCfg, substrateCfg, auditCfg *awscfg.Config) error {
	ui.Spin("configuring IAM in the audit account")

	extraAdministrator, err := policies.ExtraAdministratorAssumeRolePolicy()
	if err != nil {
		return ui.StopErr(err)
	}
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
	if err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, auditCfg, auditRole.Name, policies.AdministratorAccess); err != nil {
		return ui.StopErr(err)
	}

	// Find or create the Auditor role in the audit account. We do this even
	// if we didn't actually setup CloudTrail because having an audit account
	// with a Substrate-style Auditor role is still valuable.
	auditorAssumeRolePolicy, err := AuditorAssumeRolePolicy(ctx, mgmtCfg)
	if err != nil {
		return ui.StopErr(err)
	}
	auditorRole, err := awsiam.EnsureRole(ctx, auditCfg, roles.Auditor, auditorAssumeRolePolicy)
	if err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, auditCfg, auditorRole.Name, policies.ReadOnlyAccess); err != nil {
		return ui.StopErr(err)
	}
	allowAssumeRole, err := awsiam.EnsurePolicy(ctx, auditCfg, policies.AllowAssumeRoleName, policies.AllowAssumeRole)
	if err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, auditCfg, auditorRole.Name, aws.ToString(allowAssumeRole.Arn)); err != nil {
		return ui.StopErr(err)
	}
	denySensitiveReads, err := awsiam.EnsurePolicy(ctx, auditCfg, policies.DenySensitiveReadsName, policies.DenySensitiveReads)
	if err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.DetachRolePolicy( // was mistakenly attached up to 2023.12
		ctx,
		auditCfg,
		auditorRole.Name,
		aws.ToString(denySensitiveReads.Arn),
	); err != nil {
		return ui.StopErr(err)
	}
	//log.Print(jsonutil.MustString(auditorRole))

	return ui.StopErr(nil)
}
