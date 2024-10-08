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
	AccessControlListNotSupported = "AccessControlListNotSupported"
	AllAccessDisabled             = "AllAccessDisabled"
	BucketAlreadyExists           = "BucketAlreadyExists"
	BucketAlreadyOwnedByYou       = "BucketAlreadyOwnedByYou"
	Enabled                       = "Enabled"
	NotSignedUp                   = "NotSignedUp"
)

func EnsureBucket(
	ctx context.Context,
	cfg *awscfg.Config,
	name, region string,
	doc *policies.Document,
) (err error) {

	for {
		err = createBucket(ctx, cfg, name, region)
		if awsutil.ErrorCodeIs(err, NotSignedUp) {
			time.Sleep(1e9) // TODO exponential backoff
			continue
		}
		if awsutil.ErrorCodeIs(err, BucketAlreadyOwnedByYou) {
			err = nil
		}
		if err != nil {
			return
		}
		break
	}

	client := cfg.S3()

	// A customer once experienced a race here wherein the newly created S3
	// bucket wasn't immediately ready to have ACLs, etc. set and so failed.
	// The error was very peculiar but it's fortunately so nonsensical that
	// we can safely assume that it won't happen for any real reasons and
	// instead can be a clear sign of this race.
	for {
		_, err = client.PutBucketAcl(ctx, &s3.PutBucketAclInput{
			ACL:    types.BucketCannedACLPrivate, // the default but let's be explicit
			Bucket: aws.String(name),
		})
		if awsutil.ErrorCodeIs(err, AccessControlListNotSupported) {
			err = nil
			break
		}
		if !awsutil.ErrorCodeIs(err, AllAccessDisabled) {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	if err != nil {
		return
	}

	docJSON, err := doc.Marshal()
	if err != nil {
		return
	}
	if _, err = client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(name),
		Policy: aws.String(docJSON),
	}); err != nil {
		return
	}

	if _, err = client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(name),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
				{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
			},
		},
	}); err != nil {
		return
	}

	// TODO maybe make this optional or integrate it with a lifecycle policy on old versions?
	if _, err = client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(name),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	}); err != nil {
		return
	}

	if _, err = client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(name),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       true,
			BlockPublicPolicy:     true,
			IgnorePublicAcls:      true,
			RestrictPublicBuckets: true,
		},
	}); err != nil {
		return
	}

	return
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
