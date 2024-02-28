package awscfg

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

const (
	AWS_ACCESS_KEY_ID                = "AWS_ACCESS_KEY_ID"
	AWS_SECRET_ACCESS_KEY            = "AWS_SECRET_ACCESS_KEY"
	AWS_SESSION_TOKEN                = "AWS_SESSION_TOKEN"
	SUBSTRATE_CREDENTIALS_EXPIRATION = "SUBSTRATE_CREDENTIALS_EXPIRATION"
)

func Getenv() (creds aws.Credentials) {
	creds.AccessKeyID = os.Getenv(AWS_ACCESS_KEY_ID)
	creds.SecretAccessKey = os.Getenv(AWS_SECRET_ACCESS_KEY)
	creds.SessionToken = os.Getenv(AWS_SESSION_TOKEN)
	return
}

func Setenv(creds aws.Credentials) (err error) {
	if err = os.Setenv(AWS_ACCESS_KEY_ID, creds.AccessKeyID); err != nil {
		return
	}
	if err = os.Setenv(AWS_SECRET_ACCESS_KEY, creds.SecretAccessKey); err != nil {
		return
	}
	if creds.SessionToken == "" {
		err = os.Unsetenv(AWS_SESSION_TOKEN)
	} else {
		err = os.Setenv(AWS_SESSION_TOKEN, creds.SessionToken)
	}
	if err != nil {
		return
	}
	err = os.Setenv(SUBSTRATE_CREDENTIALS_EXPIRATION, creds.Expires.Format(time.RFC3339))
	return
}
