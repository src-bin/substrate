package awsec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/tagging"
)

type Tag = types.Tag

func CreateTags(
	ctx context.Context,
	cfg *awscfg.Config,
	resources []string,
	tags tagging.Map,
) error {
	_, err := cfg.EC2().CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: resources,
		Tags:      tagStructs(tags),
	})
	return err
}

func tagStructs(tags tagging.Map) []Tag {
	structs := make([]Tag, 0, len(tags))
	for key, value := range tags {
		structs = append(structs, Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	return structs
}
