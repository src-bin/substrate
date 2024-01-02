package humans

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/users"
)

// AdministratorAssumeRolePolicy constructs and returns an assume-role policy
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
				//roles.ARN(aws.ToString(substrateAccount.Id), roles.Administrator), // FIXME we do actually need this but can't add it on the first run
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

// AuditorAssumeRolePolicy constructs and returns an assume-role policy for
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
				//roles.ARN(aws.ToString(substrateAccount.Id), roles.Auditor), // FIXME we do actually need this but can't add it on the first run
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

// IntranetAssumeRolePolicy constructs and returns an assume-role policy for
// the Intranet to use in order to provide 12-hour sessions. The given *Config
// must be for the Substrate account or the management account.
func IntranetAssumeRolePolicy(ctx context.Context, cfg *awscfg.Config) (*policies.Document, error) {
	substrateAccount, err := cfg.FindSubstrateAccount(ctx)
	if err != nil {
		return nil, err
	}

	return policies.Merge(

		// Administrator, which is a bit of a stretch into overprivilege, but
		// is damn useful to folks debugging IAM issues.
		//
		// Note, too, that the Substrate test suite takes advantage of this
		// inclusion by the src-bin organization assuming the Administrator
		// role in various test accounts and then creating and assuming
		// roles from there.
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: []string{
			roles.ARN(aws.ToString(substrateAccount.Id), roles.Administrator),
		}}),

		// The Substrate user and role are what the Intranet actually uses
		// to move around the organization and mint 12-hour AWS credentials.
		policies.AssumeRolePolicyDocument(&policies.Principal{
			AWS: []string{
				roles.ARN(aws.ToString(substrateAccount.Id), roles.Substrate),
				users.ARN(aws.ToString(substrateAccount.Id), users.Substrate),
			},
			Service: []string{"ec2.amazonaws.com"},
		}),
	), nil
}
