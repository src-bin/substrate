package lambdautil

import (
	"bytes"
	"encoding/base64"

	"github.com/aws/aws-lambda-go/events"
)

func EventBody(event *events.APIGatewayProxyRequest) (string, error) {
	if event.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return event.Body, nil
}

func EventBody2(event *events.APIGatewayV2HTTPRequest) (string, error) {
	if event.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return event.Body, nil
}

func EventBodyBuffer(event *events.APIGatewayProxyRequest) (*bytes.Buffer, error) {
	body, err := EventBody(event)
	if err != nil {
		return nil, err
	}
	return bytes.NewBufferString(body), nil
}

func EventBodyBuffer2(event *events.APIGatewayV2HTTPRequest) (*bytes.Buffer, error) {
	body, err := EventBody2(event)
	if err != nil {
		return nil, err
	}
	return bytes.NewBufferString(body), nil
}
