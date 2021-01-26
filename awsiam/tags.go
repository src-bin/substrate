package awsiam

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/version"
)

func ListUserTags(svc *iam.IAM, userName string) (map[string]string, error) {
	var marker *string
	tags := make(map[string]string)
	for {
		in := &iam.ListUserTagsInput{
			Marker:   marker,
			UserName: aws.String(userName),
		}
		out, err := svc.ListUserTags(in)
		if err != nil {
			return nil, err
		}
		for _, tag := range out.Tags {
			tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
		}
		if !aws.BoolValue(out.IsTruncated) {
			break
		}
		marker = out.Marker
	}
	return tags, nil
}

func TagUser(svc *iam.IAM, userName, key, value string) error {
	_, err := svc.TagUser(&iam.TagUserInput{
		Tags: []*iam.Tag{
			&iam.Tag{Key: aws.String(key), Value: aws.String(value)},
		},
		UserName: aws.String(userName),
	})
	return err
}

func UntagUser(svc *iam.IAM, userName, key string) error { // make it keys ...string if needed
	_, err := svc.UntagUser(&iam.UntagUserInput{
		TagKeys:  []*string{aws.String(key)},
		UserName: aws.String(userName),
	})
	return err
}

func tagsFor(name string) []*iam.Tag {
	return []*iam.Tag{
		&iam.Tag{Key: aws.String(tags.Manager), Value: aws.String(tags.Substrate)},
		&iam.Tag{Key: aws.String(tags.SubstrateVersion), Value: aws.String(version.Version)},
	}
}
