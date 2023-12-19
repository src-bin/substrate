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

const InvalidParameterValue = "InvalidParameterValue"

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
	region0, vpcId0 string, // accepter, in Terraform's terms
	region1, vpcId1 string, // requester, in Terraform's terms
	// (I don't know why I reversed these in 2020 but now they have to stay reversed)
) (conn *VPCPeeringConnection, err error) {
	ui.Spinf("peering %s in %s with %s in %s", vpcId0, region0, vpcId1, region1)
	cfg = cfg.Regional(region1)
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
	var out *ec2.CreateVpcPeeringConnectionOutput
	if out, err = client.CreateVpcPeeringConnection(ctx, &ec2.CreateVpcPeeringConnectionInput{
		PeerRegion: aws.String(region0),
		PeerVpcId:  aws.String(vpcId0),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpcPeeringConnection,
				Tags:         tags,
			},
		},
		VpcId: aws.String(vpcId1),
	}); err == nil {
		conn = out.VpcPeeringConnection
	} else if awsutil.ErrorCodeIs(
		err,
		InvalidParameterValue,
	) && awsutil.ErrorMessageHasPrefix(
		err,
		"A matching peering exists with different tags",
	) {
		var conns []VPCPeeringConnection
		conns, err = describeVPCPeeringConnections(ctx, cfg, []types.Filter{
			{Name: aws.String("accepter-vpc-info.vpc-id"), Values: []string{vpcId0}},
			{Name: aws.String("requester-vpc-info.vpc-id"), Values: []string{vpcId1}},
		})
		if err != nil {
			ui.StopErr(err)
			return
		}
		ui.Debug(conns)
		ui.Print(err)
		if len(conns) != 1 {
			err = fmt.Errorf("expected 1 VPC peering connection but found %s", jsonutil.MustString(conns))
			ui.StopErr(err)
			return
		}
		conn = &conns[0]
		if _, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{aws.ToString(conn.VpcPeeringConnectionId)},
			Tags:      tags,
		}); err != nil {
			ui.StopErr(err)
			return
		}
	}
	ui.StopErr(err)
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
