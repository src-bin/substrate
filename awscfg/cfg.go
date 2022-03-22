package awscfg

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/telemetry"
)

type Main struct {
	cfg   aws.Config
	event *telemetry.Event
}

func NewMain(ctx context.Context) (cfg *Main, err error) {
	cfg = &Main{}
	cfg.cfg, err = config.LoadDefaultConfig(
		ctx,
		config.WithRegion(choices.DefaultRegion()),
		config.WithSharedConfigFiles(nil),
		config.WithSharedConfigProfile(""),
		config.WithSharedCredentialsFiles(nil),
	)
	if err != nil {
		return
	}
	cfg.event, err = telemetry.NewEvent(ctx)
	if err != nil {
		return
	}

	if err = cfg.organizationsTelemetry(ctx); err != nil {
		return
	}
	if err = cfg.stsTelemetry(ctx); err != nil {
		return
	}
	//log.Printf("%+v", cfg.event)

	return
}

func (cfg *Main) organizationsTelemetry(ctx context.Context) error {
	svc := organizations.NewFromConfig(cfg.cfg)
	out, err := svc.DescribeOrganization(ctx, &organizations.DescribeOrganizationInput{})
	if err != nil {
		return err
	}
	cfg.event.SetEmailDomainName(aws.ToString(out.Organization.MasterAccountEmail))
	return nil
}

func (cfg *Main) stsTelemetry(ctx context.Context) error {
	svc := sts.NewFromConfig(cfg.cfg)
	out, err := svc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return err
	}
	cfg.event.SetAccountNumber(aws.ToString(out.Account))
	if err := cfg.event.SetRoleName(aws.ToString(out.Arn)); err != nil {
		return err
	}
	return nil
}
