package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
)

type (
	SecurityGroup = types.SecurityGroup
	Subnet        = types.Subnet
	VPC           = types.Vpc
)

func DescribeSecurityGroups(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId, name string,
) ([]SecurityGroup, error) {
	out, err := cfg.EC2().DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{name},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcId},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.SecurityGroups, nil
}

func DescribeSubnets(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
) ([]Subnet, error) {
	out, err := cfg.EC2().DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcId},
		}},
	})
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.Subnets, nil
}

func DescribeVPCs(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality string,
) ([]VPC, error) {
	out, err := cfg.EC2().DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Environment"),
				Values: []string{environment},
			},
			{
				Name:   aws.String("tag:Quality"),
				Values: []string{quality},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	//log.Print(out)
	return out.Vpcs, nil
}
