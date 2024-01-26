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

type SecurityGroup = types.SecurityGroup

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

func portAuthorized[T int | int32](ports []int, port T) bool {
	for i := 0; i < len(ports); i++ {
		if ports[i] == int(port) {
			return true
		}
	}
	return false
}
