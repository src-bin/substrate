package awss3

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/version"
)

const (
	BucketAlreadyOwnedByYou = "BucketAlreadyOwnedByYou"
	Enabled                 = "Enabled"
	NotSignedUp             = "NotSignedUp"
)

func CreateBucket(svc *s3.S3, name, region string) (err error) {
	in := &s3.CreateBucketInput{
		ACL:    aws.String("private"), // the default but let's be explicit
		Bucket: aws.String(name),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		},
	}
	for {
		_, err = svc.CreateBucket(in)
		if !awsutil.ErrorCodeIs(err, NotSignedUp) {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	return
}

func EnsureBucket(svc *s3.S3, name, region string, doc *policies.Document) error {

	for {
		err := CreateBucket(svc, name, region)
		if awsutil.ErrorCodeIs(err, NotSignedUp) {
			time.Sleep(1e9) // TODO exponential backoff
			continue
		}
		if awsutil.ErrorCodeIs(err, BucketAlreadyOwnedByYou) {
			err = nil
		}
		if err != nil {
			return err
		}
		break
	}

	if _, err := svc.PutBucketAcl(&s3.PutBucketAclInput{
		ACL:    aws.String("private"), // the default but let's be explicit
		Bucket: aws.String(name),
	}); err != nil {
		return err
	}

	docJSON, err := doc.Marshal()
	if err != nil {
		return err
	}
	if _, err := svc.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: aws.String(name),
		Policy: aws.String(docJSON),
	}); err != nil {
		return err
	}

	if _, err := svc.PutBucketTagging(&s3.PutBucketTaggingInput{
		Bucket: aws.String(name),
		Tagging: &s3.Tagging{
			TagSet: []*s3.Tag{
				&s3.Tag{Key: aws.String(tags.Manager), Value: aws.String(tags.Substrate)},
				&s3.Tag{Key: aws.String(tags.SubstrateVersion), Value: aws.String(version.Version)},
			},
		},
	}); err != nil {
		return err
	}

	// TODO maybe make this optional or integrate it with a lifecycle policy on old versions?
	if _, err := svc.PutBucketVersioning(&s3.PutBucketVersioningInput{
		Bucket: aws.String(name),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: aws.String(Enabled),
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
			RestrictPublicBuckets: aws.Bool(false), // true would prevent cross-account access
		},
	}); err != nil {
		return err
	}

	return nil
}
