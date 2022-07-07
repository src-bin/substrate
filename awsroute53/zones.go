package awsroute53

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/src-bin/substrate/awscfg"
)

type HostedZone = types.HostedZone

func FindHostedZone(ctx context.Context, cfg *awscfg.Config, name string) (*HostedZone, error) {
	zones, err := ListHostedZones(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, z := range zones {
		if aws.ToString(z.Name) == name {
			zone := z // don't leak the slice
			return &zone, nil
		}
	}
	return nil, HostedZoneNotFoundError(name)
}

type HostedZoneNotFoundError string

func (err HostedZoneNotFoundError) Error() string {
	return fmt.Sprintf("HostedZoneNotFoundError: %s not found", string(err))
}

func ListHostedZones(ctx context.Context, cfg *awscfg.Config) (zones []HostedZone, err error) {
	var marker *string
	for {
		out, err := cfg.Route53().ListHostedZones(ctx, &route53.ListHostedZonesInput{Marker: marker})
		if err != nil {
			return nil, err
		}
		zones = append(zones, out.HostedZones...)
		if marker = out.NextMarker; marker == nil {
			break
		}
	}
	return
}
