package awscfg

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/telemetry"
)

const (
	TooManyRequestsException = "TooManyRequestsException"

	WaitUntilCredentialsWorkTries = 10
)

type Account = types.Account

type Config struct {
	cfg               aws.Config
	deferredTelemetry func(context.Context) error
	event             *telemetry.Event
}

func NewConfig(ctx context.Context) (c *Config, err error) {
	c = &Config{}
	c.cfg, err = config.LoadDefaultConfig(ctx, defaultLoadOptions()...)
	if err != nil {
		return
	}
	c.event, err = telemetry.NewEvent(ctx)
	if err != nil {
		return
	}

	f := func(ctx context.Context) error {
		ctx, _ = context.WithTimeout(ctx, time.Minute)
		describeOrganization, err := organizations.NewFromConfig(c.cfg).DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
		if err != nil {
			return err
		}
		c.event.SetEmailDomainName(aws.ToString(describeOrganization.Organization.MasterAccountEmail))
		getCallerIdentity, err := sts.NewFromConfig(c.cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		c.event.SetInitialAccountId(aws.ToString(getCallerIdentity.Account))
		if err := c.event.SetInitialRoleName(aws.ToString(getCallerIdentity.Arn)); err != nil {
			return err
		}
		//log.Printf("%+v", c.event)
		return nil
	}
	if err := f(ctx); err != nil {
		//log.Print(err)
		c.deferredTelemetry = f
	}

	return
}

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
	c.event.FinalAccountId = accountId
	c.event.FinalRoleName = roleName

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

func (c *Config) DescribeOrganization(ctx context.Context) (*types.Organization, error) {
	client := organizations.NewFromConfig(c.cfg)
	for {
		out, err := client.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
		/*
			if awsutil.ErrorCodeIs(err, awsorgs.AWSOrganizationsNotInUseException) {
				return c.cfg, nil
			}
		*/
		if awsutil.ErrorCodeIs(err, TooManyRequestsException) {
			time.Sleep(time.Second) // TODO exponential backoff
			continue
		}
		if err != nil {
			return nil, err
		}
		return out.Organization, nil
	}
}

func (c *Config) ClientForIAM() *iam.Client {
	return iam.NewFromConfig(c.cfg) // TODO memoize
}

func (c *Config) GetCallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	return sts.NewFromConfig(c.cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}

func (c *Config) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return c.cfg.Credentials.Retrieve(ctx)
}

func (c *Config) SetCredentials(
	ctx context.Context,
	creds aws.Credentials,
) (
	callerIdentity *sts.GetCallerIdentityOutput,
	err error,
) {
	if c.cfg, err = config.LoadDefaultConfig(
		ctx,
		loadOptions(config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{creds},
		))...,
	); err != nil {
		return
	}

	callerIdentity, err = c.WaitUntilCredentialsWork(ctx)

	if c.deferredTelemetry != nil {
		if err := c.deferredTelemetry(ctx); err == nil {
			c.deferredTelemetry = nil
		} else {
			//log.Print(err)
		}
	}
	return
}

func (c *Config) SetCredentialsV1(
	ctx context.Context,
	accessKeyId, secretAccessKey, sessionToken string,
) (*sts.GetCallerIdentityOutput, error) {
	return c.SetCredentials(ctx, aws.Credentials{
		AccessKeyID:     accessKeyId,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	})
}

func (c *Config) Telemetry() *telemetry.Event {
	return c.event
}

// WaitUntilCredentialsWork waits in a sleeping loop until the configured
// credentials (whether provided via SetCredentials or discovered in
// environment variables or an IAM instance profile) work, which it tests
// using sts:GetCallerIdentity. This seems silly but IAM is an eventually
// consistent global service so it's not guaranteed that newly created
// credentials will work immediately. Typically when this has to wait it
// waits about five seconds.
func (c *Config) WaitUntilCredentialsWork(ctx context.Context) (
	callerIdentity *sts.GetCallerIdentityOutput,
	err error,
) {
	for i := 0; i < WaitUntilCredentialsWorkTries; i++ {
		if callerIdentity, err = c.GetCallerIdentity(ctx); err == nil {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	return
}

func (c *Config) findAccount(
	ctx context.Context,
	f func(Account, tags.Tags) bool,
) (*Account, tags.Tags, error) {
	cfg, err := c.organizationReader(ctx)
	if err != nil {
		return nil, nil, err
	}

	client := organizations.NewFromConfig(cfg.cfg)
	var nextToken *string
	for {
		out, err := client.ListAccounts(ctx, &organizations.ListAccountsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, nil, err
		}
		for _, account := range out.Accounts {
			tags, err := cfg.listTagsForResource(ctx, aws.ToString(account.Id))
			if err != nil {
				return nil, nil, err
			}
			if f(account, tags) {
				return &account, tags, nil
			}
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return nil, nil, nil
}

func (c *Config) listTagsForResource(ctx context.Context, accountId string) (tags.Tags, error) {
	client := organizations.NewFromConfig(c.cfg)
	var nextToken *string
	tags := make(tags.Tags)
	for {
		out, err := client.ListTagsForResource(ctx, &organizations.ListTagsForResourceInput{
			NextToken:  nextToken,
			ResourceId: aws.String(accountId),
		})
		if err != nil {
			return nil, err
		}
		for _, tag := range out.Tags {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return tags, nil
}

func (c *Config) organizationReader(ctx context.Context) (*Config, error) {
	return c.AssumeManagementRole(ctx, roles.OrganizationReader, fmt.Sprintf(
		"%s-%s",
		contextutil.ValueString(ctx, telemetry.Command),
		contextutil.ValueString(ctx, telemetry.Subcommand),
	), time.Hour)
}

func defaultLoadOptions() []func(*config.LoadOptions) error {
	options := []func(*config.LoadOptions) error{
		config.WithSharedConfigFiles(nil),
		config.WithSharedConfigProfile(""),
		config.WithSharedCredentialsFiles(nil),
	}
	if region, err := regions.DefaultNoninteractive(); err == nil {
		options = append(options, config.WithRegion(region))
	}
	return options
}

func loadOptions(options ...func(*config.LoadOptions) error) []func(*config.LoadOptions) error {
	return append(
		defaultLoadOptions(),
		options...,
	)
}
