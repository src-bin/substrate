package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

type (
	EgressOnlyInternetGateway = types.EgressOnlyInternetGateway
	InternetGateway           = types.InternetGateway
	NATGateway                = types.NatGateway
)

func DescribeEgressOnlyInternetGateway(ctx context.Context, cfg *awscfg.Config, vpcId string) (*EgressOnlyInternetGateway, error) {
	eigws, err := DescribeEgressOnlyInternetGateways(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, eigw := range eigws {
		for _, attachment := range eigw.Attachments {
			if aws.ToString(attachment.VpcId) == vpcId {
				v := eigw // no aliasing loop variables / don't leak the whole slice
				//ui.Debug(v)
				return &v, nil
			}
		}
	}
	return nil, nil
}

func DescribeEgressOnlyInternetGateways(ctx context.Context, cfg *awscfg.Config) (gateways []EgressOnlyInternetGateway, err error) {
	var nextToken *string
	for {
		out, err := cfg.EC2().DescribeEgressOnlyInternetGateways(ctx, &ec2.DescribeEgressOnlyInternetGatewaysInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, out.EgressOnlyInternetGateways...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func DescribeInternetGateway(ctx context.Context, cfg *awscfg.Config, vpcId string) (*InternetGateway, error) {
	igws, err := DescribeInternetGateways(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, igw := range igws {
		for _, attachment := range igw.Attachments {
			if aws.ToString(attachment.VpcId) == vpcId {
				v := igw // no aliasing loop variables / don't leak the whole slice
				//ui.Debug(v)
				return &v, nil
			}
		}
	}
	return nil, nil
}

func DescribeInternetGateways(ctx context.Context, cfg *awscfg.Config) (gateways []InternetGateway, err error) {
	var nextToken *string
	for {
		out, err := cfg.EC2().DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, out.InternetGateways...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func DescribeNATGateway(ctx context.Context, cfg *awscfg.Config, publicSubnetId string) (*NATGateway, error) {
	ngws, err := DescribeNATGateways(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, ngw := range ngws {
		if aws.ToString(ngw.SubnetId) == publicSubnetId {
			v := ngw // no aliasing loop variables / don't leak the whole slice
			//ui.Debug(v)
			return &v, nil
		}
	}
	return nil, nil
}

func DescribeNATGateways(ctx context.Context, cfg *awscfg.Config) (gateways []NATGateway, err error) {
	var nextToken *string
	for {
		out, err := cfg.EC2().DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		gateways = append(gateways, out.NatGateways...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func EnsureEgressOnlyInternetGateway(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
	tags tagging.Map,
) (*EgressOnlyInternetGateway, error) {
	client := cfg.EC2()
	tags = tagging.Merge(tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateVersion: version.Version,
	}, tags)

	eigw, err := DescribeEgressOnlyInternetGateway(ctx, cfg, vpcId)
	if err != nil {
		return nil, err
	}

	if eigw == nil {
		out, err := client.CreateEgressOnlyInternetGateway(ctx, &ec2.CreateEgressOnlyInternetGatewayInput{
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeEgressOnlyInternetGateway,
					Tags:         tagStructs(tags),
				},
			},
			VpcId: aws.String(vpcId),
		})
		if err != nil {
			return nil, err
		}
		eigw = out.EgressOnlyInternetGateway
	} else {
		if err := CreateTags(ctx, cfg, []string{aws.ToString(eigw.EgressOnlyInternetGatewayId)}, tags); err != nil {
			return nil, err
		}
	}

	return eigw, nil
}

func EnsureInternetGateway(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
	tags tagging.Map,
) (*InternetGateway, error) {
	client := cfg.EC2()
	tags = tagging.Merge(tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateVersion: version.Version,
	}, tags)

	igw, err := DescribeInternetGateway(ctx, cfg, vpcId)
	if err != nil {
		return nil, err
	}

	if igw == nil {
		out, err := client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeInternetGateway,
					Tags:         tagStructs(tags),
				},
			},
		})
		if err != nil {
			return nil, err
		}
		igw = out.InternetGateway
	} else {
		if err := CreateTags(ctx, cfg, []string{aws.ToString(igw.InternetGatewayId)}, tags); err != nil {
			return nil, err
		}
	}

	if _, err := client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: igw.InternetGatewayId,
		VpcId:             aws.String(vpcId),
	}); awsutil.ErrorCodeIs(err, "Resource.AlreadyAssociated") {
		err = nil
	} else if err != nil {
		return nil, err
	}

	return igw, nil
}

func EnsureNATGateway(
	ctx context.Context,
	cfg *awscfg.Config,
	publicSubnetId string,
	tags tagging.Map,
) (*NATGateway, error) {
	client := cfg.EC2()
	tags = tagging.Merge(tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateVersion: version.Version,
	}, tags)

	ngw, err := DescribeNATGateway(ctx, cfg, publicSubnetId)
	if err != nil {
		return nil, err
	}

	if ngw == nil {

		eip, err := EnsureEIP(ctx, cfg, publicSubnetId, tags)
		if err != nil {
			return nil, err
		}

		out, err := client.CreateNatGateway(ctx, &ec2.CreateNatGatewayInput{
			AllocationId: aws.String(eip.AllocationId),
			SubnetId:     aws.String(publicSubnetId),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeNatgateway,
					Tags:         tagStructs(tags),
				},
			},
		})
		if err != nil {
			return nil, err
		}
		ngw = out.NatGateway

	} else {
		if err := CreateTags(ctx, cfg, []string{aws.ToString(ngw.NatGatewayId)}, tags); err != nil {
			return nil, err
		}
	}

	// Wait for the NAT Gateway to become available. Trying to route to one
	// that's still pending results in InvalidNatGatewayID.NotFound (wiich is
	// a confusing error) from CreateRoute.
	for range awsutil.StandardJitteredExponentialBackoff() {
		ngw, err := DescribeNATGateway(ctx, cfg, publicSubnetId)
		if err != nil {
			return nil, err
		}
		//ui.Debug(ngw)
		if ngw.State == types.NatGatewayStateAvailable {
			break
		}
	}

	return ngw, nil
}
