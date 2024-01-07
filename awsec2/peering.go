package awsec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	InvalidParameterValue             = "InvalidParameterValue"
	RouteAlreadyExists                = "RouteAlreadyExists"
	VpcPeeringConnectionAlreadyExists = "VpcPeeringConnectionAlreadyExists"
)

type VPCPeeringConnection = types.VpcPeeringConnection

func DescribeVPCPeeringConnections(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account
) ([]VPCPeeringConnection, error) {
	return describeVPCPeeringConnections(ctx, cfg, nil)
}

func EnsureVPCPeeringConnection(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account
	region0, vpcId0, region1, vpcId1 string,
) (*VPCPeeringConnection, error) {
	ui.Spinf("peering %s in %s with %s in %s", vpcId0, region0, vpcId1, region1)
	cfg = cfg.Regional(region0)
	var conn *VPCPeeringConnection

	// Try to find an existing VPC peering connection in both possible orders
	// the VPCs could be in. Either region works, though (or they're the same),
	// because, if one exists, it'll be present at both ends. This is a TOCTTOU
	// bug, technically, but ec2:CreateVpcPeeringConnection doesn't return an
	// error if the VPCs are already peered.
	for _, filters := range [][]types.Filter{
		{
			{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcId0}},
			{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcId1}},
			{Name: aws.String("status-code"), Values: []string{"active", "provisioning"}},
		},
		{
			{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcId1}},  // reversed
			{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcId0}}, // reversed
			{Name: aws.String("status-code"), Values: []string{"active", "provisioning"}},
		},
	} {
		if conns, err := describeVPCPeeringConnections(ctx, cfg, filters); err != nil {
			return nil, ui.StopErr(err)
		} else if len(conns) > 1 {
			return nil, ui.StopErr(fmt.Errorf("expected 1 VPC peering connection but found %s", jsonutil.MustString(conns)))
		} else if len(conns) == 1 {
			conn = &conns[0]
			ui.Stopf("found %s", conn.VpcPeeringConnectionId)
			return conn, nil
		}
	}

	client := cfg.EC2()
	tags := []types.Tag{
		{
			Key:   aws.String(tagging.Manager),
			Value: aws.String(tagging.Substrate),
		},
		{
			Key:   aws.String(tagging.SubstrateVersion),
			Value: aws.String(version.Version),
		},
	}
	if out, err := client.CreateVpcPeeringConnection(ctx, &ec2.CreateVpcPeeringConnectionInput{
		PeerRegion: aws.String(region1),
		PeerVpcId:  aws.String(vpcId1),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpcPeeringConnection,
				Tags:         tags,
			},
		},
		VpcId: aws.String(vpcId0),
	}); err == nil {
		conn = out.VpcPeeringConnection
		_, err = cfg.Regional(region1).EC2().AcceptVpcPeeringConnection(ctx, &ec2.AcceptVpcPeeringConnectionInput{
			VpcPeeringConnectionId: conn.VpcPeeringConnectionId,
		})
	} else if awsutil.ErrorCodeIs(
		err,
		InvalidParameterValue,
	) && awsutil.ErrorMessageHasPrefix(
		err,
		"A matching peering exists with different tags",
	) {
		ui.PrintWithCaller("tag mismatch")
		conns, err := describeVPCPeeringConnections(ctx, cfg, []types.Filter{
			{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcId0}},
			{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcId1}},
			{Name: aws.String("status-code"), Values: []string{"active", "provisioning"}},
		})
		if err != nil {
			return nil, ui.StopErr(err)
		}
		if len(conns) != 1 {
			return nil, ui.StopErr(fmt.Errorf("expected 1 VPC peering connection but found %s", jsonutil.MustString(conns)))
		}
		conn = &conns[0]
		_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{aws.ToString(conn.VpcPeeringConnectionId)},
			Tags:      tags,
		})
	}
	ui.Stopf("created %s", conn.VpcPeeringConnectionId)
	return conn, nil
}

func EnsureVPCPeeringRouteIPv4(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId, cidrPrefix, vpcPeeringConnectionId string,
) error {
	ui.Spinf("routing traffic from %s to %s via %s", routeTableId, cidrPrefix, vpcPeeringConnectionId)
	_, err := cfg.EC2().CreateRoute(ctx, &ec2.CreateRouteInput{
		DestinationCidrBlock:   aws.String(cidrPrefix),
		RouteTableId:           aws.String(routeTableId),
		VpcPeeringConnectionId: aws.String(vpcPeeringConnectionId),
	})
	if awsutil.ErrorCodeIs(err, RouteAlreadyExists) { // TODO confirm whether this detects destination gateway mismatches
		ui.Stop("route already exists")
		return nil
	}
	return ui.StopErr(err)
}

func EnsureVPCPeeringRouteIPv6(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId, cidrPrefix, vpcPeeringConnectionId string,
) error {
	ui.Spinf("routing traffic from %s to %s via %s", routeTableId, cidrPrefix, vpcPeeringConnectionId)
	_, err := cfg.EC2().CreateRoute(ctx, &ec2.CreateRouteInput{
		DestinationIpv6CidrBlock: aws.String(cidrPrefix),
		RouteTableId:             aws.String(routeTableId),
		VpcPeeringConnectionId:   aws.String(vpcPeeringConnectionId),
	})
	if awsutil.ErrorCodeIs(err, RouteAlreadyExists) { // TODO confirm whether this detects destination gateway mismatches
		ui.Stop("route already exists")
		return nil
	}
	return ui.StopErr(err)
}

func describeVPCPeeringConnections(
	ctx context.Context,
	cfg *awscfg.Config,
	filters []types.Filter,
) (conns []VPCPeeringConnection, err error) {
	client := cfg.EC2()
	var nextToken *string
	for {
		var out *ec2.DescribeVpcPeeringConnectionsOutput
		if out, err = client.DescribeVpcPeeringConnections(ctx, &ec2.DescribeVpcPeeringConnectionsInput{
			Filters:   filters,
			NextToken: nextToken,
		}); err != nil {
			return
		}
		conns = append(conns, out.VpcPeeringConnections...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
