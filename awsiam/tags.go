package awsiam

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/version"
)

func tagsFor(name string) []*iam.Tag {
	return []*iam.Tag{
		&iam.Tag{Key: aws.String("Manager"), Value: aws.String("Substrate")},
		&iam.Tag{Key: aws.String("SubstrateVersion"), Value: aws.String(version.Version)},
	}
}
