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

type VPCEndpoint = types.VpcEndpoint

func EnsureGatewayVPCEndpoint(
	ctx context.Context,
	cfg *awscfg.Config,
	vpcId string,
	routeTableIds []string,
	serviceName string, // like "com.amazonaws.us-east-1.s3"
	tags tagging.Map,
) error {
	client := cfg.EC2()
	tags = tagging.Merge(tagging.Map{
		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateVersion: version.Version,
	}, tags)
	_, err := client.CreateVpcEndpoint(ctx, &ec2.CreateVpcEndpointInput{
		RouteTableIds: routeTableIds,
		ServiceName:   aws.String(serviceName),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpcEndpoint,
				Tags:         tagStructs(tags),
			},
		},
		VpcEndpointType: types.VpcEndpointTypeGateway,
		VpcId:           aws.String(vpcId),
	})
	if awsutil.ErrorCodeIs(err, RouteAlreadyExists) {
		err = nil // if we ever actually need the *types.VpcEndpoint, we need to client.DescribeVpcEndpoints here...
	}
	if err != nil {
		return err
	}
	return nil // ...and return out.VpcEndpoint, nil here
}
