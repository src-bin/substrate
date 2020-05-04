package awsutil

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/ui"
)

func NewMasterSession(rolename string) *session.Session {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			//LogLevel:    aws.LogLevel(aws.LogDebugWithHTTPBody),
		},
		SharedConfigState: session.SharedConfigDisable,
	})
	if err != nil {
		log.Fatal(err)
	}
	out, err := organizations.New(sess).DescribeOrganization(&organizations.DescribeOrganizationInput{})
	if err != nil {
		log.Fatal(err)
	}
	return sess.Copy(&aws.Config{
		Credentials: stscreds.NewCredentials(sess, fmt.Sprintf(
			"arn:aws:iam::%s:role/%s",
			aws.StringValue(out.Organization.MasterAccountId),
			rolename,
		)),
	})
}

func NewSession(region string) *session.Session {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			//LogLevel:    aws.LogLevel(aws.LogDebugWithHTTPBody),
			Region: aws.String(region),
		},
		SharedConfigState: session.SharedConfigDisable,
	})
	if err != nil {
		log.Fatal(err)
	}
	return sess
}

func NewSessionAssumingRole(region, accountId, rolename string) *session.Session {
	return nil
}

func NewSessionExplicit(accessKeyId, secretAccessKey, region string) *session.Session {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials(accessKeyId, secretAccessKey, ""),
			//LogLevel:    aws.LogLevel(aws.LogDebugWithHTTPBody),
			Region: aws.String(region),
		},
		SharedConfigState: session.SharedConfigDisable,
	})
	if err != nil {
		log.Fatal(err)
	}
	return sess
}

func ReadAccessKeyFromStdin() (string, string) {
	accessKeyId, err := ui.Prompt("AWS access key ID:")
	if err != nil {
		log.Fatal(err)
	}
	secretAccessKey, err := ui.Prompt("AWS secret access key:")
	if err != nil {
		log.Fatal(err)
	}
	return accessKeyId, secretAccessKey
}
