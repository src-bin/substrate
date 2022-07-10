package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
)

const InvalidLaunchTemplateName_NotFoundException = "InvalidLaunchTemplateName.NotFoundException"

type (
	LaunchTemplate        = types.LaunchTemplate
	LaunchTemplateVersion = types.LaunchTemplateVersion
)

func DescribeLaunchTemplate(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
) (*LaunchTemplate, error) {
	out, err := cfg.EC2().DescribeLaunchTemplates(ctx, &ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: []string{name},
	})
	if err != nil {
		return nil, err
	}
	lt := out.LaunchTemplates[0]
	return &lt, nil
}

func DescribeLaunchTemplateVersion(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
) (*LaunchTemplateVersion, error) {
	out, err := cfg.EC2().DescribeLaunchTemplateVersions(ctx, &ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateName: aws.String(name),
		Versions:           []string{"$Latest"},
	})
	if err != nil {
		return nil, err
	}
	lt := out.LaunchTemplateVersions[0]
	return &lt, nil
}
