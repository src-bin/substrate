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
) (conn *VPCPeeringConnection, err error) {
	ui.Spinf("peering %s in %s with %s in %s", vpcId0, region0, vpcId1, region1)
	cfg = cfg.Regional(region0)
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

	// Try to find an existing VPC peering connection in both possible orders
	// the VPCs could be in. Either region works, though (or they're the same),
	// because, if one exists, it'll be present at both ends. This is a TOCTTOU
	// bug, technically, but ec2:CreateVpcPeeringConnection doesn't return an
	// error if the VPCs are already peered.
	if conn, err = describeVPCPeeringConnection(ctx, cfg, vpcId0, vpcId1); err != nil {
		return
	}

	// If we didn't find the VPC peering connection, create it.
	if conn == nil {
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
		} else if awsutil.ErrorCodeIs(
			err,
			InvalidParameterValue,
		) && awsutil.ErrorMessageHasPrefix(
			err,
			"A matching peering exists with different tags",
		) {
			if conn, err = describeVPCPeeringConnection(ctx, cfg, vpcId0, vpcId1); err != nil {
				return nil, err
			}
			_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{aws.ToString(conn.VpcPeeringConnectionId)},
				Tags:      tags,
			})
		}
	}
	//ui.Debug(conn)

	// Accept pending connections, whether created or found.
	// TODO Figure out why it appears (in the AWS Console) that connections
	// TODO _actually_ ending up in the accepted state takes a long, long time.
	// TODO It seems like it takes O(1 hour), which is just too long, though
	// TODO packets may flow sooner than the AWS Console reports that the peers
	// TODO are established.
	if conn.Status != nil && conn.Status.Code == types.VpcPeeringConnectionStateReasonCodePendingAcceptance {
		_, err = cfg.Regional(aws.ToString(conn.AccepterVpcInfo.Region)).EC2().AcceptVpcPeeringConnection(ctx, &ec2.AcceptVpcPeeringConnectionInput{
			VpcPeeringConnectionId: conn.VpcPeeringConnectionId,
		})
	}

	ui.Stop(conn.VpcPeeringConnectionId)
	return
}

func describeVPCPeeringConnection(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId0, vpcId1 string,
) (conn *VPCPeeringConnection, err error) {
	var statusCodes = []string{
		string(types.VpcPeeringConnectionStateReasonCodeActive),
		string(types.VpcPeeringConnectionStateReasonCodePendingAcceptance),
		string(types.VpcPeeringConnectionStateReasonCodeProvisioning),
	}
	for _, filters := range [][]types.Filter{
		{
			{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcId0}},
			{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcId1}},
			{Name: aws.String("status-code"), Values: statusCodes},
		},
		{
			{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcId1}},  // reversed
			{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcId0}}, // reversed
			{Name: aws.String("status-code"), Values: statusCodes},
		},
	} {
		var conns []VPCPeeringConnection
		if conns, err = describeVPCPeeringConnections(ctx, cfg, filters); err != nil {
			break
		} else if len(conns) > 1 {
			err = fmt.Errorf("expected 1 VPC peering connection but found %s", jsonutil.MustString(conns))
			break
		} else if len(conns) == 1 {
			conn = &conns[0]
			break
		}
	}
	return
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
