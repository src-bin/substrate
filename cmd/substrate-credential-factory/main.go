package main

import (
	"context"
	"errors"
	"fmt"
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

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}
	svc := sts.New(sess)

	callerIdentity, err := awssts.GetCallerIdentity(svc)
	if err != nil {
		return nil, err
	}

	principalId, ok := event.RequestContext.Authorizer["principalId"]
	if !ok {
		return nil, errors.New("could not read princpalId from Lambda request context")
	}
	sessionName, ok := principalId.(string)
	if !ok {
		return nil, fmt.Errorf("princpalId is %T not string", principalId)
	}

	assumedRole, err := awssts.AssumeRole(
		svc,
		roles.Arn(
			aws.StringValue(callerIdentity.Account),
			roles.Administrator,
		),
		sessionName,
	)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}

	body, err := lambdautil.RenderHTML(indexTemplate(), assumedRole.Credentials)
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
