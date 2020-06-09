package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/roles"
)

//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html
//go:generate go run ../../tools/template/main.go -name credentialsTemplate -package main credentials.html

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	// Serialize the event to make it available in the browser for debugging.
	b, err := json.MarshalIndent(event, "", "\t")
	if err != nil {
		return nil, err
	}

	if event.HTTPMethod == "POST" {

		sess, err := awssessions.NewSession(awssessions.Config{})
		if err != nil {
			return nil, err
		}
		svc := sts.New(sess)

		callerIdentity, err := awssts.GetCallerIdentity(svc)
		if err != nil {
			return nil, err
		}

		assumedRole, err := awssts.AssumeRole(svc, roles.Arn(
			aws.StringValue(callerIdentity.Account),
			roles.Administrator,
		))
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}

		v := struct {
			Credentials *sts.Credentials
			Debug       string
			Error       error
		}{
			Credentials: assumedRole.Credentials,
			Debug:       string(b),
		}
		body, err := lambdautil.RenderHTML(credentialsTemplate(), v)
		if err != nil {
			return nil, err
		}

		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
		}, nil
	}

	v := struct {
		Debug string
		Error error
	}{
		Debug: string(b),
	}
	body, err := lambdautil.RenderHTML(indexTemplate(), v)
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	lambda.Start(handle)
}
