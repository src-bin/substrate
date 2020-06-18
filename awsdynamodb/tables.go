package awsdynamodb

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/version"
)

const (
	PAY_PER_REQUEST        = "PAY_PER_REQUEST"
	ResourceInUseException = "ResourceInUseException"
)

func DescribeTable(
	svc *dynamodb.DynamoDB,
	name string,
) (*dynamodb.TableDescription, error) {
	out, err := svc.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	return out.Table, nil
}

func EnsureTable(
	svc *dynamodb.DynamoDB,
	name string,
	attrDefs []*dynamodb.AttributeDefinition,
	keySchema []*dynamodb.KeySchemaElement,
) (*dynamodb.TableDescription, error) {
	tags := []*dynamodb.Tag{
		&dynamodb.Tag{
			Key:   aws.String(tags.Manager),
			Value: aws.String(tags.Substrate),
		},
		&dynamodb.Tag{
			Key:   aws.String(tags.SubstrateVersion),
			Value: aws.String(version.Version),
		},
	}
	out, err := svc.CreateTable(&dynamodb.CreateTableInput{
		AttributeDefinitions: attrDefs,
		BillingMode:          aws.String(PAY_PER_REQUEST),
		KeySchema:            keySchema,
		TableName:            aws.String(name),
		Tags:                 tags,
	})
	if awsutil.ErrorCodeIs(err, ResourceInUseException) {
		out, err := svc.UpdateTable(&dynamodb.UpdateTableInput{
			AttributeDefinitions: attrDefs,
			BillingMode:          aws.String(PAY_PER_REQUEST),
			TableName:            aws.String(name),
		})
		if err != nil {
			return nil, err
		}
		//log.Printf("%+v", out)
		if _, err := svc.TagResource(&dynamodb.TagResourceInput{
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
