package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
)

func DescribeAvailabilityZones(
	ctx context.Context,
	cfg *awscfg.Config,
	region string,
) ([]types.AvailabilityZone, error) {
	out, err := cfg.EC2().DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		Filters: []types.Filter{{
			Name:   aws.String("region-name"),
			Values: []string{region},
		}},
	})
	if err != nil {
		return nil, err
	}
	return out.AvailabilityZones, nil
}
