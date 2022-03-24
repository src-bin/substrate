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
	cfg.cfg, err = config.LoadDefaultConfig(ctx, defaultLoadOptions()...)
	if err != nil {
		return
	}
	cfg.event, err = telemetry.NewEvent(ctx)
	if err != nil {
		return
	}

	}
		return
	}
	//log.Printf("%+v", cfg.event)

	return
}

func defaultLoadOptions() []func(*config.LoadOptions) error {
	return []func(*config.LoadOptions) error{
		config.WithRegion(choices.DefaultRegionNoninteractive()),
		config.WithSharedConfigFiles(nil),
		config.WithSharedConfigProfile(""),
		config.WithSharedCredentialsFiles(nil),
	}
}
