package awscloudtrail

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/version"
)

const TrailAlreadyExistsException = "TrailAlreadyExistsException"

type Trail struct {
	Arn, Name *string
}

func EnsureTrail(svc *cloudtrail.CloudTrail, name, bucketName string) (*Trail, error) {

	trail, err := createTrail(svc, name, bucketName)
	if awsutil.ErrorCodeIs(err, TrailAlreadyExistsException) {
		trail, err = updateTrail(svc, name, bucketName)
	}
	if err != nil {
		return nil, err
	}

	if _, err := svc.AddTags(&cloudtrail.AddTagsInput{
		ResourceId: trail.Arn,
		TagsList:   tagList(),
	}); err != nil {
		return nil, err
	}

	if _, err := svc.StartLogging(&cloudtrail.StartLoggingInput{
		Name: trail.Name,
	}); err != nil {
		return nil, err
	}

	return trail, nil
}

func createTrail(svc *cloudtrail.CloudTrail, name, bucketName string) (*Trail, error) {
	in := &cloudtrail.CreateTrailInput{
		EnableLogFileValidation:    aws.Bool(true),
		IncludeGlobalServiceEvents: aws.Bool(true),
		IsMultiRegionTrail:         aws.Bool(true),
		IsOrganizationTrail:        aws.Bool(true),
		Name:                       aws.String(name),
		S3BucketName:               aws.String(bucketName),
		TagsList:                   tagList(),
	}
	out, err := svc.CreateTrail(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &Trail{Arn: out.TrailARN, Name: out.Name}, nil
}

func updateTrail(svc *cloudtrail.CloudTrail, name, bucketName string) (*Trail, error) {
	in := &cloudtrail.UpdateTrailInput{
		EnableLogFileValidation:    aws.Bool(true),
		IncludeGlobalServiceEvents: aws.Bool(true),
		IsMultiRegionTrail:         aws.Bool(true),
		IsOrganizationTrail:        aws.Bool(true),
		Name:                       aws.String(name),
		S3BucketName:               aws.String(bucketName),
	}
	out, err := svc.UpdateTrail(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &Trail{Arn: out.TrailARN, Name: out.Name}, nil
}

func tagList() []*cloudtrail.Tag {
	return []*cloudtrail.Tag{
		&cloudtrail.Tag{Key: aws.String("Manager"), Value: aws.String("Substrate")},
		&cloudtrail.Tag{Key: aws.String("SubstrateVersion"), Value: aws.String(version.Version)},
	}
}
