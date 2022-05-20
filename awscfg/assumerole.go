package awscfg

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
)

func (c *Config) AssumeAdminRole(
	ctx context.Context,
	quality string,
	roleName, roleSessionName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {
	return c.AssumeServiceRole(ctx, accounts.Admin, accounts.Admin, quality, roleName, roleSessionName, duration)
}

func (c *Config) AssumeManagementRole(
	ctx context.Context,
	roleName, roleSessionName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {

	callerIdentity, err := c.GetCallerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	_ = callerIdentity
	//log.Print(jsonutil.MustString(callerIdentity))

	org, err := c.DescribeOrganization(ctx)
	if err != nil {
		return nil, err
	}
	mgmtAccountId := aws.ToString(org.MasterAccountId)
	//log.Print(jsonutil.MustString(org))
	if err := accounts.EnsureManagementAccountIdMatchesDisk(mgmtAccountId); err != nil {
		return nil, err
	}

	// TODO don't port the root user dance here, instead make that a different method

	// TODO port in the already-role/OrganizationAdministrator-or-user/OrganizationAdministrator bits

	// TODO this might not be relevant anymore but I haven't totally removed it, just in case
	/*
		// Mask the AWS-native error because we're 99% sure OrganizationReaderError
		// is a better explanation of what went wrong.
		if _, ok := err.(awserr.Error); ok { // FIXME
			ui.Fatal(awssessions.NewOrganizationReaderError(err, *roleName))
		}
	*/

	return c.AssumeRole(ctx, mgmtAccountId, roleName, roleSessionName, duration)
}

func (c *Config) AssumeRole(
	ctx context.Context,
	accountId string,
	roleName, roleSessionName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {
	if roleName != roles.OrganizationReader {
		c.event.FinalAccountId = accountId
		c.event.FinalRoleName = roleName
	}

	cfg := &Config{
		cfg:               c.cfg.Copy(),
		deferredTelemetry: c.deferredTelemetry, // better twice than not at all
		event:             c.event,
	}

	cfg.cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(
		sts.NewFromConfig(c.cfg),
		roles.Arn(accountId, roleName),
		func(options *stscreds.AssumeRoleOptions) {
			options.Duration = duration
			options.RoleSessionName = roleSessionName
		},
	))

	callerIdentity, err := cfg.WaitUntilCredentialsWork(ctx)
	_ = callerIdentity
	//log.Print(jsonutil.MustString(callerIdentity))

	return cfg, err
}

func (c *Config) AssumeServiceRole(
	ctx context.Context,
	domain, environment, quality string,
	roleName, roleSessionName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {
	account, _, err := c.findAccount(ctx, func(_ Account, t tags.Tags) bool {
		//log.Print(jsonutil.MustString(t))
		return t[tags.Domain] == domain && t[tags.Environment] == environment && t[tags.Quality] == quality
	})
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, NewAccountNotFound(domain, environment, quality)
	}
	//log.Print(jsonutil.MustString(account))
	return c.AssumeRole(ctx, aws.ToString(account.Id), roleName, roleSessionName, duration)
}

func (c *Config) AssumeSpecialRole(
	ctx context.Context,
	name string,
	roleName, roleSessionName string,
	duration time.Duration, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
) (*Config, error) {
	account, _, err := c.findAccount(ctx, func(a Account, _ tags.Tags) bool {
		//log.Print(jsonutil.MustString(a))
		return aws.ToString(a.Name) == name
	})
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, NewAccountNotFound(name)
	}
	//log.Print(jsonutil.MustString(account))
	return c.AssumeRole(ctx, aws.ToString(account.Id), roleName, roleSessionName, duration)
}
