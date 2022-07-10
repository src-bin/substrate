package awsutil

import (
	"errors"

	"github.com/aws/smithy-go"
)

const RequestError = "RequestError"

func ErrorCode(err error) string {

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
