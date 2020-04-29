package awsutil

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/src-bin/substrate/ui"
)

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
