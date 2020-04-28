package awss3

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/version"
)

const (
	BucketAlreadyOwnedByYou = "BucketAlreadyOwnedByYou"
	ENABLED                 = "ENABLED"
)

func CreateBucket(svc *s3.S3, name, region string) error {
	in := &s3.CreateBucketInput{
		ACL:    aws.String("private"), // the default but let's be explicit
		Bucket: aws.String(name),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		},
	}
	_, err := svc.CreateBucket(in)
	return err
}

func EnsureBucket(svc *s3.S3, name, region string, policy policies.Document) error {

	err := CreateBucket(svc, name, region)
	if awsutil.ErrorCodeIs(err, BucketAlreadyOwnedByYou) {
		err = nil
	}
	if err != nil {
		return err
	}

	if _, err := svc.PutBucketAcl(&s3.PutBucketAclInput{
		ACL:    aws.String("private"), // the default but let's be explicit
		Bucket: aws.String(name),
	}); err != nil {
		return err
	}

	buf, err := json.MarshalIndent(policy, "", "\t")
	if err != nil {
		return err
	}
	if _, err := svc.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: aws.String(name),
		Policy: aws.String(string(buf)),
	}); err != nil {
		return err
	}

	if _, err := svc.PutBucketTagging(&s3.PutBucketTaggingInput{
		Bucket: aws.String(name),
		Tagging: &s3.Tagging{
			TagSet: []*s3.Tag{
				&s3.Tag{Key: aws.String("Manager"), Value: aws.String("Substrate")},
				&s3.Tag{Key: aws.String("SubstrateVersion"), Value: aws.String(version.Version)},
			},
		},
	}); err != nil {
		return err
	}

	// TODO maybe make this optional or integrate it with a lifecycle policy on old versions?
	if _, err := svc.PutBucketVersioning(&s3.PutBucketVersioningInput{
		Bucket: aws.String(name),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: aws.String(ENABLED),
		},
	}); err != nil {
		return err
	}

	if _, err := svc.PutPublicAccessBlock(&s3.PutPublicAccessBlockInput{
		Bucket: aws.String(name),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(false),
		},
	}); err != nil {
		return err
	}

	return nil
}
