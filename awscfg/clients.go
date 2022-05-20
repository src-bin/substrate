package awscfg

import (
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func (c *Config) EC2() *ec2.Client { // TODO func (c *Config) EC2(region string) *ec2.Client ?
	return ec2.NewFromConfig(c.cfg) // TODO memoize
}

func (c *Config) IAM() *iam.Client {
	return iam.NewFromConfig(c.cfg) // TODO memoize
}
