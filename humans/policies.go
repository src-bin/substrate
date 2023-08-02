package humans

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/users"
)

// AdministratorAssumeRolePolicy constructs and returns an assume role policy
// for the Administrator role in any account. The given *Config must be for the
// Substrate account or the management account.
func AdministratorAssumeRolePolicy(ctx context.Context, cfg *awscfg.Config) (*policies.Document, error) {
	extraAdministrator, err := policies.ExtraAdministratorAssumeRolePolicy()
	if err != nil {
		return nil, err
	}

	mgmtAccount, err := cfg.FindManagementAccount(ctx)
	if err != nil {
		return nil, err
	}
	substrateAccount, err := cfg.FindSubstrateAccount(ctx)
	if err != nil {
		return nil, err
	}

	return policies.Merge(
		policies.AssumeRolePolicyDocument(&policies.Principal{
			AWS: []string{
				roles.ARN(aws.ToString(substrateAccount.Id), roles.Administrator),
				roles.ARN(aws.ToString(mgmtAccount.Id), roles.Substrate),
				roles.ARN(aws.ToString(substrateAccount.Id), roles.Substrate),
				users.ARN(aws.ToString(mgmtAccount.Id), users.Substrate),
				users.ARN(aws.ToString(substrateAccount.Id), users.Substrate),
			},
			Service: []string{"ec2.amazonaws.com"},
		}),
		extraAdministrator,
	), nil
}

// AuditorAssumeRolePolicy constructs and returns an assume role policy for
// the Auditor role in any account. The given *Config must be for the
// Substrate account or the management account.
func AuditorAssumeRolePolicy(ctx context.Context, cfg *awscfg.Config) (*policies.Document, error) {
	extraAdministrator, err := policies.ExtraAdministratorAssumeRolePolicy()
	if err != nil {
		return nil, err
	}
	extraAuditor, err := policies.ExtraAuditorAssumeRolePolicy()
	if err != nil {
		return nil, err
	}

	mgmtAccount, err := cfg.FindManagementAccount(ctx)
	if err != nil {
		return nil, err
	}
	substrateAccount, err := cfg.FindSubstrateAccount(ctx)
	if err != nil {
		return nil, err
	}

	return policies.Merge(
		policies.AssumeRolePolicyDocument(&policies.Principal{
			AWS: []string{
				roles.ARN(aws.ToString(substrateAccount.Id), roles.Administrator),
				roles.ARN(aws.ToString(substrateAccount.Id), roles.Auditor),
				roles.ARN(aws.ToString(mgmtAccount.Id), roles.Substrate),
				roles.ARN(aws.ToString(substrateAccount.Id), roles.Substrate),
				users.ARN(aws.ToString(mgmtAccount.Id), users.Substrate),
				users.ARN(aws.ToString(substrateAccount.Id), users.Substrate),
			},
			Service: []string{"ec2.amazonaws.com"},
		}),
		extraAdministrator,
		extraAuditor,
	), nil
}
