package awsec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cidr"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

type (
	Subnet = types.Subnet
	VPC    = types.Vpc
)

func DeleteVPC(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
) error {
	_, err := cfg.EC2().DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: aws.String(vpcId)})
	return err
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
	return out.Subnets, nil
}

func DescribeVPCs(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality string, // TODO maybe support an alternative tagging regime for the Instance Factory's VPC
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
	return out.Vpcs, nil
}

func EnsureSubnet(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
	az string,
	ipv4 cidr.IPv4,
	ipv6 cidr.IPv6,
	tags tagging.Map,
) (*Subnet, error) {
	client := cfg.EC2()
	tags = tagging.Merge(tagging.Map{
		tagging.AvailabilityZone: az,
		tagging.Manager:          tagging.Substrate,
		tagging.Region:           cfg.Region(),
		tagging.SubstrateVersion: version.Version,
	}, tags)

	out, err := client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		AvailabilityZone: aws.String(az),
		CidrBlock:        aws.String(ipv4.String()),
		Ipv6CidrBlock:    aws.String(ipv6.String()),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags:         tagStructs(tags),
			},
		},
		VpcId: aws.String(vpcId),
	})
	var subnet *Subnet
	if err == nil {
		subnet = out.Subnet
	} else if awsutil.ErrorCodeIs(err, "InvalidSubnet.Conflict") {
		subnets, err2 := DescribeSubnets(ctx, cfg, vpcId)
		if err2 != nil {
			return nil, err2
		}
		for _, s := range subnets {
			if aws.ToString(s.CidrBlock) == ipv4.String() && aws.ToString(s.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock) == ipv6.String() {
				subnet = &s
				err = nil
				if err := CreateTags(ctx, cfg, []string{aws.ToString(subnet.SubnetId)}, tags); err != nil {
					return nil, err
				}
				break
			}
		}
	}
	if err != nil {
		return nil, err
	}

	if _, err := client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		AssignIpv6AddressOnCreation: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		SubnetId:                    subnet.SubnetId,
	}); err != nil {
		return nil, err
	}
	if _, err := client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		EnableResourceNameDnsAAAARecordOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		SubnetId:                                subnet.SubnetId,
	}); err != nil {
		return nil, err
	}
	if _, err := client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		EnableResourceNameDnsARecordOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(true)},
		SubnetId:                             subnet.SubnetId,
	}); err != nil {
		return nil, err
	}
	if _, err := client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		MapPublicIpOnLaunch: &types.AttributeBooleanValue{Value: aws.Bool(tags[tagging.Connectivity] == "public")},
		SubnetId:            subnet.SubnetId,
	}); err != nil {
		return nil, err
	}
	if _, err := client.ModifySubnetAttribute(ctx, &ec2.ModifySubnetAttributeInput{
		PrivateDnsHostnameTypeOnLaunch: types.HostnameTypeResourceName,
		SubnetId:                       subnet.SubnetId,
	}); err != nil {
		return nil, err
	}

	return subnet, nil
}

func EnsureVPC(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality string,
	ipv4 cidr.IPv4,
	tags tagging.Map,
) (*VPC, error) {
	tags = tagging.Merge(tagging.Map{
		tagging.Environment:      environment,
		tagging.Manager:          tagging.Substrate,
		tagging.Name:             fmt.Sprintf("%s-%s", environment, quality),
		tagging.Quality:          quality,
		tagging.SubstrateVersion: version.Version,
	}, tags)

	vpcs, err := DescribeVPCs(ctx, cfg, environment, quality)
	if err != nil {
		return nil, err
	}

	if len(vpcs) == 0 {
		return createVPC(ctx, cfg, environment, quality, ipv4, tags)
	}
	if len(vpcs) != 1 { // TODO support sharing many VPCs when we introduce `substrate network create|delete|list`
		return nil, fmt.Errorf("expected 1 VPC but found %s", jsonutil.MustString(vpcs))
	}
	if aws.ToString(vpcs[0].CidrBlock) != ipv4.String() {
		return nil, fmt.Errorf(
			"expected VPC with CIDR prefix %s but found CIDR prefix %s",
			aws.ToString(vpcs[0].CidrBlock),
			ipv4.String(),
		)
	}

	if err := CreateTags(ctx, cfg, []string{aws.ToString(vpcs[0].VpcId)}, tags); err != nil {
		return nil, err
	}

	return &vpcs[0], nil
}

func createVPC(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality string,
	ipv4 cidr.IPv4,
	tags tagging.Map,
) (*VPC, error) {
	client := cfg.EC2()
	out, err := client.CreateVpc(ctx, &ec2.CreateVpcInput{
		AmazonProvidedIpv6CidrBlock: aws.Bool(true),
		CidrBlock:                   aws.String(ipv4.String()),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags:         tagStructs(tags),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if _, err := client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(true)}, // must come first, by itself
		VpcId:            out.Vpc.VpcId,
	}); err != nil {
		return nil, err
	}
	if _, err := client.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(true)}, // must come second, also by itself
		VpcId:              out.Vpc.VpcId,
	}); err != nil {
		return nil, err
	}
	return out.Vpc, nil
}
