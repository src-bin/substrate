package awss3

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func CreateBucket(svc *s3.S3, name, region string) (*s3.CreateBucketOutput, error) {
	in := &s3.CreateBucketInput{
		ACL:    aws.String("private"), // the default but let's be explicit
		Bucket: aws.String(name),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(region),
		},
	}
	out, err := svc.CreateBucket(in)
	if err != nil {
		return nil, err
	}
	log.Printf("%+v", out)
	return out, nil
}

func EnsureBucket(svc *s3.S3, name, region, policy string) (*s3.Bucket, error) {
	_, err := CreateBucket(svc, name, region)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
