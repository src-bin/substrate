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
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/terraform"
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
		tagging.Environment: environment,
		tagging.Quality:     quality,
		tagging.Region:      region,

		tagging.Manager:          tagging.Substrate,
		tagging.SubstrateVersion: version.Version,
	}
	if domain == naming.Admin {
		tags[tagging.Name] = fmt.Sprintf("%s-%s", domain, quality) // special case for the Substrate account
	} else {
		tags[tagging.Name] = fmt.Sprintf("%s-%s-%s", domain, environment, quality)
	}

	// Find the VPC and subnets to share.
	vpcs, err := awsec2.DescribeVPCs(ctx, networkCfg, environment, quality)
	ui.Must(err)
	if len(vpcs) != 1 { // TODO support sharing many VPCs when we introduce `substrate create-network` and friends
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

// StateRm removes the resource share, shared subnets, and all the tags from
// Terraform to stop it from trying to manage them.
func StateRm(dirname, domain, environment, quality, region string) {
	if !fileutil.IsDir(dirname) {
		return
	}
	tfTags := terraform.Tags{
		Domain:      domain,
		Environment: environment,
		Quality:     quality,
		Region:      region,
	}
	ui.Spinf("removing VPC sharing resources from Terraform in %s", dirname)
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ec2_tag.%s", terraform.Label(tfTags, "subnet-connectivity"))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ec2_tag.%s", terraform.Label(tfTags, "subnet-environment"))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ec2_tag.%s", terraform.Label(tfTags, "subnet-name"))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ec2_tag.%s", terraform.Label(tfTags, "subnet-quality"))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ec2_tag.%s", terraform.Label(tfTags, "vpc-environment"))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ec2_tag.%s", terraform.Label(tfTags, "vpc-name"))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ec2_tag.%s", terraform.Label(tfTags, "vpc-quality"))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ram_principal_association.%s", terraform.Label(tfTags))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ram_resource_association.%s", terraform.Label(tfTags))))
	ui.Must(terraform.StateRm(dirname, fmt.Sprintf("aws_ram_resource_share.%s", terraform.Label(tfTags))))
	ui.Stop("ok")
}
