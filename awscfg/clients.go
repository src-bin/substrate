package awscfg

import (
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func (c *Config) APIGateway() *apigateway.Client {
	return apigateway.NewFromConfig(c.cfg) // TODO memoize regionally
}

func (c *Config) CloudTrail() *cloudtrail.Client {
	return cloudtrail.NewFromConfig(c.cfg) // TODO memoize regionally
}

func (c *Config) DynamoDB() *dynamodb.Client {
	return dynamodb.NewFromConfig(c.cfg) // TODO memoize regionally
}

func (c *Config) EC2() *ec2.Client {
	return ec2.NewFromConfig(c.cfg) // TODO memoize regionally
}

func (c *Config) IAM() *iam.Client {
	return iam.NewFromConfig(c.cfg) // TODO memoize
}

func (c *Config) Organizations() *organizations.Client {
	return organizations.NewFromConfig(c.cfg) // TODO memoize
}

func (c *Config) RAM() *ram.Client {
	return ram.NewFromConfig(c.cfg) // TODO memoize (regionally?)
}

func (c *Config) Route53() *route53.Client {
	return route53.NewFromConfig(c.cfg) // TODO memoize
}

func (c *Config) S3() *s3.Client {
	return s3.NewFromConfig(c.cfg) // TODO memoize regionally
}

func (c *Config) STS() *sts.Client {
	return sts.NewFromConfig(c.cfg) // TODO memoize
}

func (c *Config) SecretsManager() *secretsmanager.Client {
	return secretsmanager.NewFromConfig(c.cfg) // TODO memoize regionally
}

func (c *Config) ServiceQuotas() *servicequotas.Client {
	return servicequotas.NewFromConfig(c.cfg) // TODO memoize regionally
}
