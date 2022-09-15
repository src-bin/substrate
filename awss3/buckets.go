package awss3

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

const (
	BucketAlreadyOwnedByYou = "BucketAlreadyOwnedByYou"
	Enabled                 = "Enabled"
	NotSignedUp             = "NotSignedUp"
)

func EnsureBucket(
	ctx context.Context,
	cfg *awscfg.Config,
	name, region string,
	doc *policies.Document,
) error {

	for {
		err := createBucket(ctx, cfg, name, region)
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

	client := cfg.S3()

	if _, err := client.PutBucketAcl(ctx, &s3.PutBucketAclInput{
		ACL:    types.BucketCannedACLPrivate, // the default but let's be explicit
		Bucket: aws.String(name),
	}); err != nil {
		return err
	}

	docJSON, err := doc.Marshal()
	if err != nil {
		return err
	}
	if _, err := client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(name),
		Policy: aws.String(docJSON),
	}); err != nil {
		return err
	}

	if _, err := client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(name),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
				{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
			},
		},
	}); err != nil {
		return err
	}

	// TODO maybe make this optional or integrate it with a lifecycle policy on old versions?
	if _, err := client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(name),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	}); err != nil {
		return err
	}

	if _, err := client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(name),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       true,
			BlockPublicPolicy:     true,
			IgnorePublicAcls:      true,
			RestrictPublicBuckets: true,
		},
	}); err != nil {
		return err
	}

	return nil
}

func createBucket(ctx context.Context, cfg *awscfg.Config, name, region string) (err error) {
	in := &s3.CreateBucketInput{
		ACL:    types.BucketCannedACLPrivate, // the default but let's be explicit
		Bucket: aws.String(name),
	}
	if region != "us-east-1" { // can't specify this, can only let it default to this
		in.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}
	for {
		_, err = cfg.S3().CreateBucket(ctx, in)
		if !awsutil.ErrorCodeIs(err, NotSignedUp) {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	return
}
