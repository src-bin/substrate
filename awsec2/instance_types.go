package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
)

type (
	InstanceTypeInfo     = types.InstanceTypeInfo
	InstanceTypeOffering = types.InstanceTypeOffering
	InstanceType         = types.InstanceType
)

func DescribeInstanceTypeOfferings(
	ctx context.Context,
	cfg *awscfg.Config,
) (offerings []InstanceTypeOffering, err error) {
	var nextToken *string
	for {
		out, err := cfg.EC2().DescribeInstanceTypeOfferings(ctx, &ec2.DescribeInstanceTypeOfferingsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		offerings = append(offerings, out.InstanceTypeOfferings...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func DescribeInstanceTypes(
	ctx context.Context,
	cfg *awscfg.Config,
	instanceTypes []InstanceType,
) (infos []InstanceTypeInfo, err error) {
	var nextToken *string
	for {
		out, err := cfg.EC2().DescribeInstanceTypes(ctx, &ec2.DescribeInstanceTypesInput{
			InstanceTypes: instanceTypes,
			NextToken:     nextToken,
		})
		if err != nil {
			return nil, err
		}
		infos = append(infos, out.InstanceTypes...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
