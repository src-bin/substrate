package awsram

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ram/types"
	"github.com/src-bin/substrate/tagging"
)

type Tag = types.Tag

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
