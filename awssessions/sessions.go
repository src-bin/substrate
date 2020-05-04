package awssessions

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
)

func AccessKeyCredentials(accessKeyId, secretAccessKey string) *credentials.Credentials {
	return credentials.NewStaticCredentials(accessKeyId, secretAccessKey, "")
}

func AssumeRole(sess *session.Session, accountId, rolename string) *session.Session {
	return sess.Copy(&aws.Config{
		Credentials: stscreds.NewCredentials(sess, fmt.Sprintf(
			"arn:aws:iam::%s:role/%s",
			accountId,
			rolename,
		)),
	})
}

func AssumeRoleMaster(sess *session.Session, rolename string) *session.Session {
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if err != nil {
		log.Fatal(err)
	}
	return AssumeRole(sess, aws.StringValue(org.MasterAccountId), rolename)
}

func Config() *aws.Config {
	return &aws.Config{
		//LogLevel: aws.LogLevel(aws.LogDebugWithHTTPBody),
	}
}

func NewSession(config *aws.Config) *session.Session {
	return session.Must(session.NewSessionWithOptions(options(config)))
}

func options(config *aws.Config) session.Options {
	return session.Options{
		Config:            *config,
		SharedConfigState: session.SharedConfigDisable,
	}
}
