package networks

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsec2"
	"github.com/src-bin/substrate/awsram"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func ShareVPC(
	ctx context.Context,
	accountCfg, networkCfg *awscfg.Config,
	domain, environment, quality string,
	region string,
) {
	ui.Spinf("sharing the %s/%s VPC with account %s", environment, quality, accountCfg.MustAccountId(ctx))

	// Mimic exactly what we were doing in Terraform for a smooth transition.
	tags := tagging.Map{
		tagging.Name: nameTag(domain, environment, quality),

		tagging.Environment: environment,
		tagging.Quality:     quality,

		tagging.Region: region,

		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateVersion: version.Version,
	}

	// Find the VPC and subnets to share.
	vpcs, err := awsec2.DescribeVPCs(ctx, networkCfg, environment, quality)
	ui.Must(err)
	if len(vpcs) != 1 { // TODO support sharing many VPCs when we introduce `substrate network create|delete|list`
		ui.Fatalf("expected 1 VPC but found %s", jsonutil.MustString(vpcs))
	}
	vpc := vpcs[0]
	subnets, err := awsec2.DescribeSubnets(ctx, networkCfg, aws.ToString(vpc.VpcId))
	ui.Must(err)

	// Find or create a Resource Share in the network account and ensure it
	// shares at least these subnets with at least this service account.
	var resources []string
	for _, subnet := range subnets {
		resources = append(resources, aws.ToString(subnet.SubnetArn))
	}
	ui.Must2(awsram.EnsureResourceShare(
		ctx,
		networkCfg,
		tags[tagging.Name],
		[]string{accountCfg.MustAccountId(ctx)}, // principals
		resources,
		tags,
	))

	// Tag the shared subnets in the service account since tags don't propagate
	// when resources are shared.
	for _, subnet := range subnets {
		for range awsutil.StandardJitteredExponentialBackoff() {
			_, err := accountCfg.EC2().CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{aws.ToString(subnet.SubnetId)},
				Tags:      subnet.Tags,
			})
			if err == nil {
				break
			} else if !awsutil.ErrorCodeIs(err, "InvalidSubnetID.NotFound") {
				ui.Fatal(err)
			}
		}
	}
	ui.Must2(accountCfg.EC2().CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{aws.ToString(vpc.VpcId)},
		Tags:      vpc.Tags,
	}))

	ui.Stop("ok")
}

func nameTag(domain, environment, quality string) string {
	if domain == naming.Admin {
		return fmt.Sprintf("%s-%s", domain, quality) // special case for the Substrate account
	}
	return fmt.Sprintf("%s-%s-%s", domain, environment, quality)
}
