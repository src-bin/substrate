package awscfg

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/telemetry"
)

type Main struct {
	cfg               aws.Config
	deferredTelemetry func() error
	event             *telemetry.Event
}

func NewMain(ctx context.Context) (cfg *Main, err error) {
	cfg = &Main{}
	cfg.cfg, err = config.LoadDefaultConfig(ctx, defaultLoadOptions()...)
	if err != nil {
		return
	}
	cfg.event, err = telemetry.NewEvent(ctx)
	if err != nil {
		return
	}

	f := func() error {
		describeOrganization, err := organizations.NewFromConfig(cfg.cfg).DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
		if err != nil {
			return err
		}
		cfg.event.SetEmailDomainName(aws.ToString(describeOrganization.Organization.MasterAccountEmail))
		getCallerIdentity, err := sts.NewFromConfig(cfg.cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}
		cfg.event.SetInitialAccountNumber(aws.ToString(getCallerIdentity.Account))
		if err := cfg.event.SetInitialRoleName(aws.ToString(getCallerIdentity.Arn)); err != nil {
			return err
		}
		//log.Printf("%+v", cfg.event)
		return nil
	}
	if err := f(); err != nil {
		//log.Print(err)
		cfg.deferredTelemetry = f
	}

	return
}

func (cfg *Main) SetCredentials(ctx context.Context, creds *aws.Credentials) (err error) {
	if cfg.cfg, err = config.LoadDefaultConfig(
		ctx,
		loadOptions(config.WithCredentialsProvider(credentials.StaticCredentialsProvider{*creds}))...,
	); err != nil {
		return
	}

	if cfg.deferredTelemetry != nil {
		err = cfg.deferredTelemetry()
	}
	cfg.deferredTelemetry = nil
	return
}

func (cfg *Main) SetCredentialsV1(ctx context.Context, accessKeyId, secretAccessKey, sessionToken string) error {
	return cfg.SetCredentials(ctx, &aws.Credentials{
		AccessKeyID:     accessKeyId,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	})
}

func (cfg *Main) Telemetry() *telemetry.Event {
	return cfg.event
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
