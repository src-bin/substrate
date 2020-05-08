package awssessions

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

type Config struct {
	AccessKeyId, SecretAccessKey string
	Region                       string
}

func (c Config) AWS() aws.Config {
	var a aws.Config

	if c.AccessKeyId != "" && c.SecretAccessKey != "" {
		a.Credentials = credentials.NewStaticCredentials(c.AccessKeyId, c.SecretAccessKey, "")
	}
	if c.AccessKeyId != "" && c.SecretAccessKey == "" {
		ui.Print("ignoring access key ID without secret access key")
	}
	if c.AccessKeyId == "" && c.SecretAccessKey != "" {
		ui.Print("ignoring secret access key without access key ID")
	}

	if c.Region != "" {
		a.Region = aws.String(c.Region)
	}

	return a
}

func AssumeRole(sess *session.Session, accountId, rolename string) *session.Session {
	return sess.Copy(&aws.Config{
		Credentials: stscreds.NewCredentials(sess, roles.ARN(accountId, rolename)),
	})
}

func AssumeRoleMaster(sess *session.Session, rolename string) *session.Session {
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if err != nil {
		log.Fatal(err)
	}
	return AssumeRole(sess, aws.StringValue(org.MasterAccountId), rolename)
}

func NewSession(config Config) *session.Session {
	return session.Must(session.NewSessionWithOptions(options(config.AWS())))
}

func options(config aws.Config) session.Options {
	return session.Options{
		Config:            config,
		SharedConfigState: session.SharedConfigDisable,
	}
}
