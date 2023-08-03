package humans

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
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
