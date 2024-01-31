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
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

type RouteTable = types.RouteTable

func CreateRouteTable(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId, subnetId string,
	tags tagging.Map,
) (*RouteTable, error) {
	client := cfg.EC2()
	tags = tagging.Merge(tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateVersion: version.Version,
	}, tags)

	out, err := client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeRouteTable,
				Tags:         tagStructs(tags),
			},
		},
		VpcId: aws.String(vpcId),
	})
	if err != nil {
		return nil, err
	}

	if _, err := client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: out.RouteTable.RouteTableId,
		SubnetId:     aws.String(subnetId),
	}); err != nil {
		return nil, err
	}

	return out.RouteTable, nil
}

func DeleteRouteIPv4(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId string,
	ipv4 cidr.IPv4,
) error {
	ui.Printf("dropping route for traffic from %s to %s", routeTableId, ipv4)
	_, err := cfg.EC2().DeleteRoute(ctx, &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String(ipv4.String()),
		RouteTableId:         aws.String(routeTableId),
	})
	if awsutil.ErrorCodeIs(err, "InvalidRoute.NotFound") {
		err = nil
	}
	return err
}

func DescribeRouteTables(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
) (
	public *RouteTable,
	private map[string]*RouteTable, // subnetId to RouteTable
	err error,
) {
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
	private = make(map[string]*RouteTable)
	for _, rt := range out.RouteTables {
		//ui.Debug(rt)
		v := rt // no aliasing loop variables / don't leak the whole slice
		for _, assoc := range rt.Associations {
			if aws.ToBool(assoc.Main) {
				public = &v
			} else {
				private[aws.ToString(assoc.SubnetId)] = &v
			}
		}
	}
	return
}

func EnsureEgressOnlyInternetGatewayRouteIPv6(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId string,
	ipv6 cidr.IPv6,
	egressOnlyInternetGatewayId string,
) error {
	ui.Printf("routing traffic from %s to %s via %s", routeTableId, ipv6, egressOnlyInternetGatewayId)
	return ensureRoute(ctx, cfg, routeTableId, &ec2.CreateRouteInput{
		DestinationIpv6CidrBlock:    aws.String(ipv6.String()),
		EgressOnlyInternetGatewayId: aws.String(egressOnlyInternetGatewayId),
	})
}

func EnsureInternetGatewayRouteIPv4(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId string,
	ipv4 cidr.IPv4,
	internetGatewayId string,
) error {
	ui.Printf("routing traffic from %s to %s via %s", routeTableId, ipv4, internetGatewayId)
	return ensureRoute(ctx, cfg, routeTableId, &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(ipv4.String()),
		GatewayId:            aws.String(internetGatewayId),
	})
}

func EnsureInternetGatewayRouteIPv6(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId string,
	ipv6 cidr.IPv6,
	internetGatewayId string,
) error {
	ui.Printf("routing traffic from %s to %s via %s", routeTableId, ipv6, internetGatewayId)
	return ensureRoute(ctx, cfg, routeTableId, &ec2.CreateRouteInput{
		DestinationIpv6CidrBlock: aws.String(ipv6.String()),
		GatewayId:                aws.String(internetGatewayId),
	})
}

func EnsureNATGatewayRouteIPv4(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId string,
	ipv4 cidr.IPv4,
	natGatewayId string,
) error {
	ui.Printf("routing traffic from %s to %s via %s", routeTableId, ipv4, natGatewayId)
	return ensureRoute(ctx, cfg, routeTableId, &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(ipv4.String()),
		NatGatewayId:         aws.String(natGatewayId),
	})
}

func EnsureVPCPeeringRouteIPv4(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId string,
	ipv4 cidr.IPv4,
	vpcPeeringConnectionId string,
) error {
	ui.Printf("routing traffic from %s to %s via %s", routeTableId, ipv4, vpcPeeringConnectionId)
	return ensureRoute(ctx, cfg, routeTableId, &ec2.CreateRouteInput{
		DestinationCidrBlock:   aws.String(ipv4.String()),
		VpcPeeringConnectionId: aws.String(vpcPeeringConnectionId),
	})
}

func EnsureVPCPeeringRouteIPv6(
	ctx context.Context,
	cfg *awscfg.Config, // must be in the network account and in the right region
	routeTableId string,
	ipv6 cidr.IPv6,
	vpcPeeringConnectionId string,
) error {
	ui.Printf("routing traffic from %s to %s via %s", routeTableId, ipv6, vpcPeeringConnectionId)
	return ensureRoute(ctx, cfg, routeTableId, &ec2.CreateRouteInput{
		DestinationIpv6CidrBlock: aws.String(ipv6.String()),
		VpcPeeringConnectionId:   aws.String(vpcPeeringConnectionId),
	})
}

func ensureRoute(ctx context.Context, cfg *awscfg.Config, routeTableId string, in *ec2.CreateRouteInput) error {
	in.RouteTableId = aws.String(routeTableId)
	_, err := cfg.EC2().CreateRoute(ctx, in)
	if awsutil.ErrorCodeIs(err, RouteAlreadyExists) { // TODO confirm whether this detects destination gateway mismatches
		err = nil
	}
	return err
}
