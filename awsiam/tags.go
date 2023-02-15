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

func TagRole(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
	tags tagging.Map,
) error {
	_, err := cfg.IAM().TagRole(ctx, &iam.TagRoleInput{
		RoleName: aws.String(roleName),
		Tags:     tagStructs(tags),
	})
	return err
}

func TagUser(
	ctx context.Context,
	cfg *awscfg.Config,
	userName string,
	tags tagging.Map,
) error {
	_, err := cfg.IAM().TagUser(ctx, &iam.TagUserInput{
		Tags:     tagStructs(tags),
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

func tagStructs(tags tagging.Map) []types.Tag {
	structs := make([]types.Tag, 0, len(tags))
	for key, value := range tags {
		structs = append(structs, types.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return structs
}
