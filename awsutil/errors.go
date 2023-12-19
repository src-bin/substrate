package awsutil

import (
	"errors"
	"strings"

	"github.com/aws/smithy-go"
)

const RequestError = "RequestError"

func ErrorCode(err error) string {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		return ae.ErrorCode()
	}
	return ""
}

func ErrorCodeIs(err error, code string) bool {
	return ErrorCode(err) == code
}

func ErrorMessage(err error) string {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		return ae.ErrorMessage()
	}
	return err.Error()
}

func ErrorMessageHasPrefix(err error, prefix string) bool {
	return strings.HasPrefix(ErrorMessage(err), prefix)
}
