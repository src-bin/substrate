package awsutil

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/src-bin/substrate/ui"
)

func NewSession() *session.Session {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	return sess
}

func NewSessionExplicit(accessKeyId, secretAccessKey string) *session.Session {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials(accessKeyId, secretAccessKey, ""),
			//LogLevel:    aws.LogLevel(aws.LogDebugWithHTTPBody),
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
