package awscfg

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/ui"
)

const TooManyRequestsException = "TooManyRequestsException"

type (
	Account struct {
		types.Account
		Tags tags.Tags
	}
	Organization = types.Organization
)

type Config struct {
	cfg                     aws.Config
	deferredTelemetry       func(context.Context) error
	event                   *telemetry.Event
	getCallerIdentityOutput *sts.GetCallerIdentityOutput // cache
	organization            *Organization                // cache
}

func Must(cfg *Config, err error) *Config {
	if err != nil {
		ui.Fatal(err)
	}
	return cfg
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
		describeOrganization, err := c.Organizations().DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
		if err != nil {
			return err
		}
		c.event.SetEmailDomainName(aws.ToString(describeOrganization.Organization.MasterAccountEmail))
		getCallerIdentity, err := c.STS().GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
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
	ctx1s, _ := context.WithTimeout(ctx, time.Second)
	if err := f(ctx1s); err != nil {
		//log.Print(err)
		c.deferredTelemetry = f
	}

	return
}

func (c *Config) Copy() *Config {
	c2 := *c
	return &c2
}

func (c *Config) DescribeOrganization(ctx context.Context) (*Organization, error) {
	if c.organization != nil {
		return c.organization, nil
	}
	client := c.Organizations()
	for {
		out, err := client.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
		if awsutil.ErrorCodeIs(err, TooManyRequestsException) {
			time.Sleep(time.Second) // TODO exponential backoff
			continue
		}
		if err != nil {
			return nil, err
		}
		c.organization = out.Organization
		return out.Organization, nil
	}
}

func (c *Config) GetCallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	if c.getCallerIdentityOutput != nil {
		return c.getCallerIdentityOutput, nil
	}
	out, err := c.STS().GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	c.getCallerIdentityOutput = out
	return out, nil
}

func (c *Config) MustDescribeOrganization(ctx context.Context) *Organization {
	org, err := c.DescribeOrganization(ctx)
	if err != nil {
		ui.Fatal(err)
	}
	return org
}

func (c *Config) MustGetCallerIdentity(ctx context.Context) *sts.GetCallerIdentityOutput {
	callerIdentity, err := c.GetCallerIdentity(ctx)
	if err != nil {
		ui.Fatal(err)
	}
	return callerIdentity
}

func (c *Config) OrganizationReader(ctx context.Context) (*Config, error) {
	// TODO return early if we're already OrganizationReader
	return c.AssumeManagementRole(ctx, roles.OrganizationReader, time.Hour)
}

func (c *Config) Region() string {
	return c.cfg.Region
}

func (c *Config) Regional(region string) *Config {
	c2 := c.Copy()
	c2.cfg.Region = region
	return c2
}

func (c *Config) Tags(ctx context.Context) (map[string]string, error) {
	callerIdentity, err := c.GetCallerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	cfg, err := c.OrganizationReader(ctx)
	if err != nil {
		return nil, err
	}
	return cfg.listTagsForResource(ctx, aws.ToString(callerIdentity.Account))
}

func (c *Config) Telemetry() *telemetry.Event {
	return c.event
}

func (c *Config) listTagsForResource(ctx context.Context, accountId string) (tags.Tags, error) {
	client := c.Organizations()
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

func defaultLoadOptions() []func(*config.LoadOptions) error {
	options := []func(*config.LoadOptions) error{
		config.WithRetryMaxAttempts(10),
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
