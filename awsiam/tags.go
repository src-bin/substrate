package awsiam

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/version"
)

func tagsFor(name string) []*iam.Tag {
	return []*iam.Tag{
		&iam.Tag{Key: aws.String(tags.Manager), Value: aws.String(tags.Substrate)},
		&iam.Tag{Key: aws.String(tags.SubstrateVersion), Value: aws.String(version.Version)},
	}
}
