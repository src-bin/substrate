package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

//go:generate go run ../../tools/template/main.go -name authorizeTemplate -package main authorize.html
//go:generate go run ../../tools/template/main.go -name indexTemplate -package main index.html

const (
	CreateAccessKeyTriesBeforeDeleteAll = 4
	CreateAccessKeyTriesTotal           = 8
)

func getCredentials(sess *session.Session, sessionName string) (*sts.Credentials, error) {
	iamSvc := iam.New(sess)
	var (
		accessKey *iam.AccessKey
		err       error
	)
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
		return nil, err
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
		return nil, err
	}

	return assumedRole.Credentials, nil
}

func getSessionName(event *events.APIGatewayProxyRequest) (string, error) {
	principalId, ok := event.RequestContext.Authorizer["principalId"]
	if !ok {
		return "", errors.New("could not read princpalId from Lambda request context")
	}
	sessionName, ok := principalId.(string)
	if !ok {
		return "", fmt.Errorf("princpalId is %T not string", principalId)
	}
	return sessionName, nil
}

func handle(ctx context.Context, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	// TODO arrange some kind of garbage collection of stale tags

	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	switch event.Path {
	case "/credential-factory":

		sessionName, err := getSessionName(event)
		if err != nil {
			return nil, err
		}
		credentials, err := getCredentials(sess, sessionName)
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}

		body, err := lambdautil.RenderHTML(indexTemplate(), credentials)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
		}, nil

	case "/credential-factory/authorize":

		// Tag the CredentialFactory IAM user using the bearer token as the key and
		// the session name as the value. We choose to use tags as our database here
		// because there won't be that many and it's free. We choose to use tags on
		// an IAM resource because they're global and Substrate's Intranet is
		// multi-region.
		sessionName, err := getSessionName(event)
		if err != nil {
			return nil, err
		}
		token, ok := event.QueryStringParameters["token"]
		if !ok {
			return lambdautil.ErrorResponse(errors.New("query string parameter ?token= is required"))
		}
		var _ = sessionName // TODO tag users.CredentialFactory with token=sessionName

		body, err := lambdautil.RenderHTML(authorizeTemplate(), token)
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayProxyResponse{
			Body:       body,
			Headers:    map[string]string{"Content-Type": "text/html"},
			StatusCode: http.StatusOK,
		}, nil

	case "/credential-factory/fetch":

		// Requests to this endpoint are not authenticated or authorized by API
		// Gateway. Instead, we authorize requests by their presentation of a
		// valid bearer token. Validity is determined by finding a matching tag
		// on the CredentialFactory IAM user.
		token, ok := event.QueryStringParameters["token"]
		if !ok {
			return lambdautil.ErrorResponse(errors.New("query string parameter ?token= is required"))
		}
		var _ = token          // TODO check the tag; untag and 200 if found; 403 if not
		var sessionName string // TODO replace with the tag value
		if true {
			return &events.APIGatewayProxyResponse{
				Body:       `{"status": "403 Forbidden"}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
				StatusCode: http.StatusForbidden,
			}, nil
		}

		// HERE BE DRAGONS
		// If execution reaches this point without proper authorization then very
		// privileged AWS credentials will be leaked to whomever made the request.

		credentials, err := getCredentials(sess, sessionName)
		if err != nil {
			return lambdautil.ErrorResponse(err)
		}
		body, err := json.MarshalIndent(credentials, "", "\t")
		if err != nil {
			return nil, err
		}
		return &events.APIGatewayProxyResponse{
			Body:       string(body),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: http.StatusOK,
		}, nil

	default:
		return &events.APIGatewayProxyResponse{
			Body:       "404 Not Found\n",
			Headers:    map[string]string{"Content-Type": "text/plain"},
			StatusCode: http.StatusNotFound,
		}, nil
	}

}

func main() {
	lambda.Start(handle)
}
