package awscloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

const ResourceAlreadyExistsException = "ResourceAlreadyExistsException"

func EnsureLogGroup(ctx context.Context, cfg *awscfg.Config, name string, retention /* in days */ int) (err error) {
	client := cfg.CloudWatchLogs()

	_, err = client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(name),
	})
	if awsutil.ErrorCodeIs(err, ResourceAlreadyExistsException) {
		err = nil
	}
	if err != nil {
		return
	}

	_, err = client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(name),
		RetentionInDays: aws.Int32(int32(retention)),
	})
	if awsutil.ErrorCodeIs(err, ResourceAlreadyExistsException) {
		err = nil
	}
	if err != nil {
		return
	}

	return
}
