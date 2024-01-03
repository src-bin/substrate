package awscfg

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/ui"
)

const (
	CachedOrganizationFilename = ".substrate.organization.json" // cached on disk (obviously)

	TooManyRequestsException    = "TooManyRequestsException"
	UnrecognizedClientException = "UnrecognizedClientException"
)

type Config struct {
	accounts                []*Account // cache
	accountsExpiry          time.Time  // cache expiry
	cfg                     aws.Config
	deferredTelemetry       func(context.Context) error
	event                   *telemetry.Event
	getCallerIdentityOutput *sts.GetCallerIdentityOutput // cache
	organization            *Organization                // cache
	wd                      string                       // detect os.Chdir to bust cache
}

func Must(cfg *Config, err error) *Config {
	if err != nil {
		ui.Fatal(err)
	}
	return cfg
}

func NewConfig(ctx context.Context) (c *Config, err error) {
	c = &Config{}
	if c.cfg, err = config.LoadDefaultConfig(ctx, defaultLoadOptions()...); err != nil {
		return
	}
	if c.event, err = telemetry.NewEvent(ctx); err != nil {
		return
	}
	if c.wd, err = os.Getwd(); err != nil {
		return
	}
	//ui.PrintfWithCaller("c.wd: %s", c.wd)

	f := func(ctx context.Context) error {
		chOrg, chErr := make(chan *Organization), make(chan error)
		go func() {
			org, err := c.DescribeOrganization(ctx)
			chOrg <- org
			chErr <- err
		}()
		callerIdentity, err := c.GetCallerIdentity(ctx)
		if err != nil {
			return err
		}
		org, err := <-chOrg, <-chErr
		if err != nil {
			return err
		}

		c.event.SetEmailDomainName(aws.ToString(org.MasterAccountEmail))
		if ss := strings.Split(aws.ToString(callerIdentity.UserId), ":"); len(ss) > 1 { // e.g. "AROASTEM43Z77S3GZP5PP:rcrowley@src-bin.com"
			c.event.SetEmailSHA256(ss[1])
		}
		c.event.SetInitialAccountId(aws.ToString(callerIdentity.Account))
		if err := c.event.SetInitialRoleName(aws.ToString(callerIdentity.Arn)); err != nil {
			return err
		}
		//ui.PrintfWithCaller("%+v", c.event)
		return nil
	}
	ctx1s, _ := context.WithTimeout(ctx, time.Second)
	if err := f(ctx1s); err != nil {
		//ui.PrintWithCaller(err)
		c.deferredTelemetry = f
	}

	return
}

func (c *Config) AccountId(ctx context.Context) (string, error) {
	callerIdentity, err := c.GetCallerIdentity(ctx)
	if err != nil {
		return "", err
	}
	return aws.ToString(callerIdentity.Account), nil
}

func (c *Config) Copy() *Config {
	c2 := *c
	return &c2
}

func (c *Config) DescribeOrganization(ctx context.Context) (*Organization, error) {

	if c.organization != nil {
		return c.organization, nil
	}

	if pathname, err := fileutil.PathnameInParents(CachedOrganizationFilename); err == nil {
		if err := jsonutil.Read(pathname, &c.organization); err == nil {
			return c.organization, nil
		}
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
		break
	}

	if wd, err := os.Getwd(); err == nil && c.wd == wd {
		if pathname, err := fileutil.PathnameInParents(AccountsFilename); err == nil {
			if err := EnsureManagementAccountIdMatchesDisk(aws.ToString(c.organization.MasterAccountId)); err == nil {
				pathname := filepath.Join(filepath.Dir(pathname), CachedOrganizationFilename)
				if err := jsonutil.Write(c.organization, pathname); err != nil {
					ui.Print(err)
				}
			}
		}
	}

	return c.organization, nil
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

func (c *Config) MustAccountId(ctx context.Context) string {
	accountId, err := c.AccountId(ctx)
	ui.Must(err)
	return accountId
}

func (c *Config) MustDescribeOrganization(ctx context.Context) *Organization {
	org, err := c.DescribeOrganization(ctx)
	ui.Must(err)
	return org
}

func (c *Config) MustGetCallerIdentity(ctx context.Context) *sts.GetCallerIdentityOutput {
	callerIdentity, err := c.GetCallerIdentity(ctx)
	ui.Must(err)
	return callerIdentity
}

func (c *Config) OrganizationReader(ctx context.Context) (*Config, error) {
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

func (c *Config) Tags(ctx context.Context) (tagging.Map, error) {
	callerIdentity, err := c.GetCallerIdentity(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := c.listCachedAccounts()
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if aws.ToString(account.Id) == aws.ToString(callerIdentity.Account) {
			return account.Tags, nil
		}
	}

	cfg, err := c.OrganizationReader(ctx) // TODO sometimes takes more than 1s!
	if err != nil {
		return nil, err
	}
	return cfg.listTagsForResource(ctx, aws.ToString(callerIdentity.Account))
}

func (c *Config) Telemetry() *telemetry.Event {
	return c.event
}

func (c *Config) listTagsForResource(ctx context.Context, accountId string) (tagging.Map, error) {
	client := c.Organizations()
	var nextToken *string
	tags := make(tagging.Map)
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

type Organization = types.Organization

func defaultLoadOptions() []func(*config.LoadOptions) error {

	// The AWS SDK defaults to 9 retries (10 total tries) for most API requests
	// and 2 retries (3 total tries) twice (using two different strategies) for
	// requests to IMDS (169.254.169.254). Those are all crazy high and, more
	// importantly, ruin some folks' experience on Macs when for some reason
	// connecting to 169.254.169.254 takes a long, long time (multiple minutes)
	// to not work.
	const defaultRetries = 3

	i, err := strconv.Atoi(os.Getenv("SUBSTRATE_DEBUG_AWS_RETRIES"))
	if err != nil {
		i = defaultRetries
	}
	if i == 0 {
		ui.Printf("configuring the AWS SDK to not retry per SUBSTRATE_DEBUG_AWS_RETRIES", i)
	} else if i != defaultRetries {
		ui.Printf(
			"configuring the AWS SDK to retry up to %d times instead of the default %d per SUBSTRATE_DEBUG_AWS_RETRIES",
			i,
			defaultRetries,
		)
	}
	options := []func(*config.LoadOptions) error{
		config.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = i + 1
				o.Retryables = append(o.Retryables, retry.IsErrorRetryableFunc(func(err error) aws.Ternary {
					if awsutil.ErrorCodeIs(err, UnrecognizedClientException) {
						return aws.TrueTernary
					}
					return aws.UnknownTernary
				}))
			})
		}),
		config.WithSharedConfigFiles([]string{}),
		config.WithSharedConfigProfile(""),
		config.WithSharedCredentialsFiles([]string{}),
	}
	if os.Getenv("SUBSTRATE_DEBUG_AWS_LOGS") != "" {
		options = append(
			options,
			config.WithClientLogMode(aws.LogRequestWithBody|aws.LogResponseWithBody|aws.LogRetries),
		)
		ui.Print("configuring the AWS SDK to log request and response bodies per SUBSTRATE_DEBUG_AWS_LOGS")
	}

	// Sometimes, for reasons we don't quite understand, 169.254.169.254 takes
	// a long time to fail on a Mac which makes literally every Substrate
	// program much slower than necessary. To avoid this, we no longer make
	// any attempt to use the EC2 IMDS on a Mac. (Realistically no one is ever
	// going to use Substrate on one of those fancy EC2 Macs so this is fine.)
	if runtime.GOOS == "darwin" {
		options = append(
			options,
			config.WithEC2IMDSClientEnableState(imds.ClientDisabled),
		)
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
