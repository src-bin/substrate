package awsutil

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/smithy-go"
)

const RequestError = "RequestError"

func ErrorCode(err error) string {

	// AWS SDK for Go v1
	if e, ok := err.(awserr.Error); ok {
		return e.Code()
	}

	// AWS SDK for Go v2
	var ae smithy.APIError
	if errors.As(err, &ae) {
		return ae.ErrorCode()
	}

	return ""
}

func ErrorCodeIs(err error, code string) bool {
	return ErrorCode(err) == code
}
