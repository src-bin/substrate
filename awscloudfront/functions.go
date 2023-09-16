package awscloudfront

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

const FunctionAlreadyExists = "FunctionAlreadyExists"

type EventType = types.EventType

const (
	OriginRequest  = types.EventTypeOriginRequest
	OriginResponse = types.EventTypeOriginResponse
	ViewerRequest  = types.EventTypeViewerRequest
	ViewerResponse = types.EventTypeViewerResponse
)

func ensureFunction(ctx context.Context, cfg *awscfg.Config, name, code string) (functionARN string, err error) {
	client := cfg.CloudFront()
	functionConfig := &types.FunctionConfig{
		Comment: aws.String(name),
		Runtime: types.FunctionRuntimeCloudfrontJs20,
	}
	var out *cloudfront.CreateFunctionOutput
	out, err = client.CreateFunction(ctx, &cloudfront.CreateFunctionInput{
		FunctionCode:   []byte(code),
		FunctionConfig: functionConfig,
		Name:           aws.String(name),
	})
	var etag *string
	if err == nil {
		etag = out.ETag
		functionARN = aws.ToString(out.FunctionSummary.FunctionMetadata.FunctionARN)
	} else if awsutil.ErrorCodeIs(err, FunctionAlreadyExists) {
		{
			var out *cloudfront.DescribeFunctionOutput
			if out, err = client.DescribeFunction(ctx, &cloudfront.DescribeFunctionInput{
				Name: aws.String(name),
			}); err != nil {
				return
			}
			etag = out.ETag
			functionARN = aws.ToString(out.FunctionSummary.FunctionMetadata.FunctionARN)
		}
		{
			var out *cloudfront.UpdateFunctionOutput
			if out, err = client.UpdateFunction(ctx, &cloudfront.UpdateFunctionInput{
				FunctionCode:   []byte(code),
				FunctionConfig: functionConfig,
				IfMatch:        etag,
				Name:           aws.String(name),
			}); err != nil {
				return
			}
			etag = out.ETag
		}
	}
	_, err = client.PublishFunction(ctx, &cloudfront.PublishFunctionInput{
		IfMatch: etag,
		Name:    aws.String(name),
	})
	return
}
