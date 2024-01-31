package awsec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

type EIP struct {
	AllocationId, IPv4 string
}

func DescribeEIP(
	ctx context.Context,
	cfg *awscfg.Config,
	ownerTagValue string, // like a secondary index for this EIP
) (*EIP, error) {
	out, err := cfg.EC2().DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{{
			Name:   aws.String("tag:Owner"),
			Values: []string{ownerTagValue},
		}},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Addresses) == 0 {
		return nil, nil
	}
	if len(out.Addresses) > 1 {
		return nil, fmt.Errorf("expected 1 EIP but found %s", jsonutil.MustString(out.Addresses))
	}
	return &EIP{
		AllocationId: aws.ToString(out.Addresses[0].AllocationId),
		IPv4:         aws.ToString(out.Addresses[0].PublicIp),
	}, nil
}

func EnsureEIP(
	ctx context.Context,
	cfg *awscfg.Config,
	ownerTagValue string, // like a secondary index for this EIP
	tags tagging.Map,
) (*EIP, error) {
	client := cfg.EC2()
	tags = tagging.Merge(tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.Owner:            ownerTagValue,
		tagging.SubstrateVersion: version.Version,
	}, tags)

	eip, err := DescribeEIP(ctx, cfg, ownerTagValue)
	if err != nil {
		return nil, err
	}

	if eip == nil {
		out, err := client.AllocateAddress(ctx, &ec2.AllocateAddressInput{
			Domain: types.DomainTypeVpc,
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeElasticIp,
					Tags:         tagStructs(tags),
				},
			},
		})
		if err != nil {
			return nil, err
		}
		eip = &EIP{
			AllocationId: aws.ToString(out.AllocationId),
			IPv4:         aws.ToString(out.PublicIp),
		}
	} else {
		if err := CreateTags(ctx, cfg, []string{eip.AllocationId}, tags); err != nil {
			return nil, err
		}
	}

	return eip, nil
}
