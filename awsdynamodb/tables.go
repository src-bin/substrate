package awsdynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

const ResourceInUseException = "ResourceInUseException"

type (
	AttributeDefinition = types.AttributeDefinition
	KeySchemaElement    = types.KeySchemaElement
	TableDescription    = types.TableDescription
)

func DescribeTable(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
) (*TableDescription, error) {
	out, err := cfg.DynamoDB().DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	return out.Table, nil
}

func EnsureTable(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	attrDefs []AttributeDefinition,
	keySchema []KeySchemaElement,
) (*TableDescription, error) {
	client := cfg.DynamoDB()
	tags := []types.Tag{
		{
			Key:   aws.String(tagging.Manager),
			Value: aws.String(tagging.Substrate),
		},
		{
			Key:   aws.String(tagging.SubstrateVersion),
			Value: aws.String(version.Version),
		},
	}
	out, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		AttributeDefinitions: attrDefs,
		BillingMode:          types.BillingModePayPerRequest,
		KeySchema:            keySchema,
		TableName:            aws.String(name),
		Tags:                 tags,
	})
	if awsutil.ErrorCodeIs(err, ResourceInUseException) {
		out, err := client.UpdateTable(ctx, &dynamodb.UpdateTableInput{
			AttributeDefinitions: attrDefs,
			BillingMode:          types.BillingModePayPerRequest,
			TableName:            aws.String(name),
		})
		if err != nil {
			return nil, err
		}
		//log.Printf("%+v", out)
		if _, err := client.TagResource(ctx, &dynamodb.TagResourceInput{
			ResourceArn: out.TableDescription.TableArn,
			Tags:        tags,
		}); err != nil {
			return nil, err
		}
		return out.TableDescription, nil
	}
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.TableDescription, nil
}
