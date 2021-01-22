package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/users"
)

//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html

const (
	CreateAccessKeyTriesBeforeDeleteAll = 4
	CreateAccessKeyTriesTotal           = 8
)

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	principalId, ok := event.RequestContext.Authorizer["principalId"]
	if !ok {
		return nil, errors.New("could not read princpalId from Lambda request context")
	}
	sessionName, ok := principalId.(string)
	if !ok {
		return nil, fmt.Errorf("princpalId is %T not string", principalId)
	}

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}
	iamSvc := iam.New(sess)

	var accessKey *iam.AccessKey
	for i := 0; i < CreateAccessKeyTriesTotal; i++ {
		accessKey, err = awsiam.CreateAccessKey(iamSvc, users.CredentialFactory)
		if awsutil.ErrorCodeIs(err, awsiam.LimitExceeded) {
			if i == CreateAccessKeyTriesBeforeDeleteAll {
				if err := awsiam.DeleteAllAccessKeys(iamSvc, users.CredentialFactory); err != nil {
					return nil, err
				}
			}
			continue
		}
		break
	}
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}
	defer func() {
		if err := awsiam.DeleteAccessKey(
			iamSvc,
			users.CredentialFactory,
			aws.StringValue(accessKey.AccessKeyId),
		); err != nil {
			log.Print(err)
		}
	}()

	time.Sleep(3e9) // I really wish I didn't have to do this

	userSess, err := awssessions.NewSession(awssessions.Config{
		AccessKeyId:     aws.StringValue(accessKey.AccessKeyId),
		SecretAccessKey: aws.StringValue(accessKey.SecretAccessKey),
	})
	if err != nil {
		return nil, err
	}
	stsSvc := sts.New(userSess)

	callerIdentity, err := awssts.GetCallerIdentity(stsSvc)
	if err != nil {
		return nil, err
	}
	assumedRole, err := awssts.AssumeRole(
		stsSvc,
		roles.Arn(
			aws.StringValue(callerIdentity.Account),
			roles.Administrator, // TODO parameterize by user as with AWS Console
		),
		sessionName,
		43200,
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
