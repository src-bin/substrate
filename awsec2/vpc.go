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
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

type (
	RouteTable    = types.RouteTable
	SecurityGroup = types.SecurityGroup
	Subnet        = types.Subnet
	VPC           = types.Vpc
)

func CreateVPC(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality string, // TODO maybe support an alternative tagging regime for the Instance Factory's VPC
	cidrPrefix cidr.IPv4,
) (*VPC, error) {
	client := cfg.EC2()
	out, err := client.CreateVpc(ctx, &ec2.CreateVpcInput{
		AmazonProvidedIpv6CidrBlock: aws.Bool(true),
		CidrBlock:                   aws.String(cidrPrefix.String()),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: tagStructs(tagging.Map{
					tagging.Environment:      environment,
					tagging.Manager:          tagging.Substrate,
					tagging.Name:             fmt.Sprintf("%s-%s", environment, quality),
					tagging.Quality:          quality,
					tagging.SubstrateVersion: version.Version,
				}),
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

func DeleteVPC(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
) error {
	_, err := cfg.EC2().DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: aws.String(vpcId)})
	return err
}

func DescribeRouteTables(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
) (public *RouteTable, private []RouteTable, err error) {
	var out *ec2.DescribeRouteTablesOutput
	if out, err = cfg.EC2().DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcId},
		}},
		MaxResults: aws.Int32(5), // any more than 4 will be an error
	}); err != nil {
		return
	}
	if len(out.RouteTables) > 4 {
		err = fmt.Errorf("found too many routing tables in %s", vpcId)
		return
	}
	for _, rt := range out.RouteTables {
		//ui.Debug(rt)
		count := 0
		for _, assoc := range rt.Associations {
			if aws.ToBool(assoc.Main) {
				continue
			}
			if assoc.SubnetId != nil {
				count++
			}
		}
		switch count {
		case 1:
			private = append(private, rt)
		case 3:
			publicValue := rt
			public = &publicValue
		default:
			ui.Print("found unexpected routing table ", jsonutil.MustOneLineString(rt))
		}
	}
	return
}

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

func EnsureSecurityGroup(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId, name string,
	tcpIngressPorts []int, // TODO support more protocols and finer-grained sources as needed
) (*SecurityGroup, error) {
	client := cfg.EC2()
	_, err := cfg.EC2().CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		Description: aws.String(name),
		GroupName:   aws.String(name),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags: []types.Tag{
					{
						Key:   aws.String(tagging.Manager),
						Value: aws.String(tagging.Substrate),
					},
					{
						Key:   aws.String(tagging.SubstrateVersion),
						Value: aws.String(version.Version),
					},
				},
			},
		},
		VpcId: aws.String(vpcId),
	})
	if err != nil && !awsutil.ErrorCodeIs(err, "InvalidGroup.Duplicate") {
		return nil, err
	}

	securityGroups, err := DescribeSecurityGroups(ctx, cfg, vpcId, name)
	if err != nil {
		return nil, err
	}

	var ipPermissions []types.IpPermission
	for _, p := range securityGroups[0].IpPermissions {
		if aws.ToInt32(p.FromPort) != aws.ToInt32(p.ToPort) {
			continue
		}
		if len(p.IpRanges) != 1 && aws.ToString(p.IpRanges[0].CidrIp) != "0.0.0.0/0" {
			continue
		}
		if len(p.Ipv6Ranges) != 1 && aws.ToString(p.Ipv6Ranges[0].CidrIpv6) != "::/0" {
			continue
		}
		if portAuthorized(tcpIngressPorts, aws.ToInt32(p.FromPort)) {
			continue
		}
		ipPermissions = append(ipPermissions, p)
	}
	//ui.Debug(ipPermissions)
	if len(ipPermissions) > 0 {
		if _, err := client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       securityGroups[0].GroupId,
			IpPermissions: ipPermissions,
		}); err != nil {
			return nil, err
		}
	}

	for _, port := range tcpIngressPorts {
		if _, err := client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: securityGroups[0].GroupId,
			IpPermissions: []types.IpPermission{{ // one at a time to tolerate duplicate errors
				FromPort:   aws.Int32(int32(port)),
				IpProtocol: aws.String("tcp"),
				IpRanges:   []types.IpRange{{CidrIp: aws.String("0.0.0.0/0")}},
				Ipv6Ranges: []types.Ipv6Range{{CidrIpv6: aws.String("::/0")}},
				ToPort:     aws.Int32(int32(port)),
			}},
		}); err != nil && !awsutil.ErrorCodeIs(err, "InvalidPermission.Duplicate") {
			return nil, err
		}
	}

	// TODO more complex ingress rules
	// TODO egress rules of any sort

	if securityGroups, err = DescribeSecurityGroups(ctx, cfg, vpcId, name); err != nil {
		return nil, err
	}
	return &securityGroups[0], nil
}

func EnsureVPC(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality string, // TODO maybe support an alternative tagging regime for the Instance Factory's VPC
	cidrPrefix cidr.IPv4,
) (*VPC, error) {
	vpcs, err := DescribeVPCs(ctx, cfg, environment, quality)
	if err != nil {
		return nil, err
	}
	if len(vpcs) == 0 {
		return CreateVPC(ctx, cfg, environment, quality, cidrPrefix)
	}
	if len(vpcs) != 1 { // TODO support sharing many VPCs when we introduce `substrate network create|delete|list`
		return nil, fmt.Errorf("expected 1 VPC but found %s", jsonutil.MustString(vpcs))
	}
	if aws.ToString(vpcs[0].CidrBlock) != cidrPrefix.String() {
		return nil, fmt.Errorf(
			"expected VPC with CIDR prefix %s but found CIDR prefix %s",
			aws.ToString(vpcs[0].CidrBlock),
			cidrPrefix.String(),
		)
	}
	if err := CreateTags(ctx, cfg, []string{aws.ToString(vpcs[0].VpcId)}, tagging.Map{
		tagging.Environment:      environment,
		tagging.Manager:          tagging.Substrate,
		tagging.Name:             fmt.Sprintf("%s-%s", environment, quality),
		tagging.Quality:          quality,
		tagging.SubstrateVersion: version.Version,
	}); err != nil {
		return nil, err
	}
	return &vpcs[0], nil
}

func portAuthorized[T int | int32](ports []int, port T) bool {
	for i := 0; i < len(ports); i++ {
		if ports[i] == int(port) {
			return true
		}
	}
	return false
}
