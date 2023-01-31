package awscloudtrail

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

const TrailAlreadyExistsException = "TrailAlreadyExistsException"

type Trail = types.Trail

type TrailDescriptor struct { // because the various APIs don't all use types.Trail
	TrailARN, Name *string
}

func DescribeTrails(ctx context.Context, cfg *awscfg.Config) ([]Trail, error) {
	out, err := cfg.CloudTrail().DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{
		IncludeShadowTrails: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	//log.Print(jsonutil.MustString(out))
	return out.TrailList, nil
}

func EnsureTrail(ctx context.Context, cfg *awscfg.Config, name, bucketName string) (*TrailDescriptor, error) {

	trail, err := createTrail(ctx, cfg, name, bucketName)
	if awsutil.ErrorCodeIs(err, TrailAlreadyExistsException) {
		trail, err = updateTrail(ctx, cfg, name, bucketName)
	}
	if err != nil {
		return nil, err
	}

	client := cfg.CloudTrail()

	if _, err := client.AddTags(ctx, &cloudtrail.AddTagsInput{
		ResourceId: trail.TrailARN,
		TagsList:   tagList(),
	}); err != nil {
		return nil, err
	}

	if _, err := client.StartLogging(ctx, &cloudtrail.StartLoggingInput{
		Name: trail.Name,
	}); err != nil {
		return nil, err
	}

	return trail, nil
}

func createTrail(ctx context.Context, cfg *awscfg.Config, name, bucketName string) (*TrailDescriptor, error) {
	out, err := cfg.CloudTrail().CreateTrail(ctx, &cloudtrail.CreateTrailInput{
		EnableLogFileValidation:    aws.Bool(true),
		IncludeGlobalServiceEvents: aws.Bool(true),
		IsMultiRegionTrail:         aws.Bool(true),
		IsOrganizationTrail:        aws.Bool(true),
		Name:                       aws.String(name),
		S3BucketName:               aws.String(bucketName),
		TagsList:                   tagList(),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &TrailDescriptor{TrailARN: out.TrailARN, Name: out.Name}, nil
}

func updateTrail(ctx context.Context, cfg *awscfg.Config, name, bucketName string) (*TrailDescriptor, error) {
	out, err := cfg.CloudTrail().UpdateTrail(ctx, &cloudtrail.UpdateTrailInput{
		EnableLogFileValidation:    aws.Bool(true),
		IncludeGlobalServiceEvents: aws.Bool(true),
		IsMultiRegionTrail:         aws.Bool(true),
		IsOrganizationTrail:        aws.Bool(true),
		Name:                       aws.String(name),
		S3BucketName:               aws.String(bucketName),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &TrailDescriptor{TrailARN: out.TrailARN, Name: out.Name}, nil
}

func tagList() []types.Tag {
	return []types.Tag{
		{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
		{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
	}
}
