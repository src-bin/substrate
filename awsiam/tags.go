package awsiam

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

func ListUserTags(
	ctx context.Context,
	cfg *awscfg.Config,
	userName string,
) (tagging.Map, error) {
	var marker *string
	tags := make(tagging.Map)
	for {
		out, err := cfg.IAM().ListUserTags(ctx, &iam.ListUserTagsInput{
			Marker:   marker,
			UserName: aws.String(userName),
		})
		if err != nil {
			return nil, err
		}
		for _, tag := range out.Tags {
			tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		if !out.IsTruncated {
			break
		}
		marker = out.Marker
	}
	return tags, nil
}

func TagUser(
	ctx context.Context,
	cfg *awscfg.Config,
	userName, key, value string,
) error {
	_, err := cfg.IAM().TagUser(ctx, &iam.TagUserInput{
		Tags: []types.Tag{
			{Key: aws.String(key), Value: aws.String(value)},
		},
		UserName: aws.String(userName),
	})
	return err
}

func UntagUser(
	ctx context.Context,
	cfg *awscfg.Config,
	userName, key string,
) error { // make it keys ...string if needed
	_, err := cfg.IAM().UntagUser(ctx, &iam.UntagUserInput{
		TagKeys:  []string{key},
		UserName: aws.String(userName),
	})
	return err
}

func tagsFor(name string) []types.Tag {
	return []types.Tag{
		{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
		{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
	}
}
