package awsiam

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	iamv1 "github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/version"
)

func ListUserTags(
	ctx context.Context,
	cfg *awscfg.Config,
	userName string,
) (map[string]string, error) {
	var marker *string
	tags := make(map[string]string)
	for {
		out, err := cfg.ClientForIAM().ListUserTags(ctx, &iam.ListUserTagsInput{
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
	_, err := cfg.ClientForIAM().TagUser(ctx, &iam.TagUserInput{
		Tags: []types.Tag{
			{Key: aws.String(key), Value: aws.String(value)},
		},
		UserName: aws.String(userName),
	})
	return err
}

func TagUserV1(
	svc *iamv1.IAM,
	userName, key, value string,
) error {
	_, err := svc.TagUser(&iamv1.TagUserInput{
		Tags: []*iamv1.Tag{
			&iamv1.Tag{Key: aws.String(key), Value: aws.String(value)},
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
	_, err := cfg.ClientForIAM().UntagUser(ctx, &iam.UntagUserInput{
		TagKeys:  []string{key},
		UserName: aws.String(userName),
	})
	return err
}

func tagsFor(name string) []types.Tag {
	return []types.Tag{
		{Key: aws.String(tags.Manager), Value: aws.String(tags.Substrate)},
		{Key: aws.String(tags.SubstrateVersion), Value: aws.String(version.Version)},
	}
}

func tagsForV1(name string) []*iamv1.Tag {
	return []*iamv1.Tag{
		&iamv1.Tag{Key: aws.String(tags.Manager), Value: aws.String(tags.Substrate)},
		&iamv1.Tag{Key: aws.String(tags.SubstrateVersion), Value: aws.String(version.Version)},
	}
}
