package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/src-bin/substrate/lambdautil"
)

func main() {
	lambda.Start(start)
}

func start(ctx context.Context, event *lambdautil.ProxyEvent) (*lambdautil.ProxyResponse, error) {
	b, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return nil, err
	}
	return &lambdautil.ProxyResponse{
		Body:       "We did it!\n\n" + string(b),
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusOK,
	}, nil
}
