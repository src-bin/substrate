package awsutil

import "github.com/aws/aws-sdk-go/aws/awserr"

func ErrorCode(err error) string {
	if e, ok := err.(awserr.Error); ok {
		return e.Code()
	}
	return ""
}

func ErrorCodeIs(err error, code string) bool {
	return ErrorCode(err) == code
}
