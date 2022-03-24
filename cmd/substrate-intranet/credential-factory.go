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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/users"
)

//go:generate go run ../../tools/template/main.go -name credentialFactoryAuthorizeTemplate -package main credential-factory-authorize.html
//go:generate go run ../../tools/template/main.go -name credentialFactoryTemplate -package main credential-factory.html

const (
	CreateAccessKeyTriesBeforeDeleteAll = 4
	CreateAccessKeyTriesTotal           = 8

	MinTokenLength = 40

	TagKeyPrefix   = "CredentialFactory:"
	TagValueFormat = "%s %s expiry %s"
)

type TagValue struct {
	Expiry                time.Time
	PrincipalId, RoleName string
}

func NewTagValue(principalId, roleName string) *TagValue {
	return &TagValue{
		Expiry:      time.Now().Add(time.Minute),
		PrincipalId: principalId,
		RoleName:    roleName,
	}
}

func ParseTagValue(raw string) (*TagValue, error) {
	if raw == "" {
		return nil, errors.New("empty tag value")
	}
	var s string
	v := &TagValue{}
	_, err := fmt.Sscanf(raw, TagValueFormat, &v.PrincipalId, &v.RoleName, &s)
	if err != nil {
		return nil, err
	}
	v.Expiry, err = time.Parse(time.RFC3339, s)
	return v, err
}

func (v *TagValue) Expired() bool {
	return time.Now().After(v.Expiry)
}

func (v *TagValue) String() string {
	return fmt.Sprintf(
		TagValueFormat,
		v.PrincipalId,
		v.RoleName,
		v.Expiry.Format(time.RFC3339),
	)
}

func credentialFactoryHandler(ctx context.Context, cfg *awscfg.Main, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	credentials, err := getCredentials(
		sess,
		event.RequestContext.Authorizer[authorizerutil.PrincipalId].(string),
		event.RequestContext.Authorizer[authorizerutil.RoleName].(string),
	)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}

	body, err := lambdautil.RenderHTML(credentialFactoryTemplate(), credentials)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil
}

func credentialFactoryAuthorizeHandler(ctx context.Context, cfg *awscfg.Main, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}

	// TODO arrange some kind of garbage collection of stale tags

	// Tag the CredentialFactory IAM user using the bearer token as the key and
	// the session name as the value. We choose to use tags as our database here
	// because there won't be that many and it's free. We choose to use tags on
	// an IAM resource because they're global and Substrate's Intranet is
	// multi-region.
	token, ok := event.QueryStringParameters["token"]
	if !ok {
		return lambdautil.ErrorResponse(errors.New("query string parameter token is required"))
	}
	if len(token) < MinTokenLength {
		return lambdautil.ErrorResponse(fmt.Errorf("token must be at least %d characters long", MinTokenLength))
	}
	if err := awsiam.TagUser(
		iam.New(sess),
		users.CredentialFactory,
		TagKeyPrefix+token,
		NewTagValue(
			event.RequestContext.Authorizer[authorizerutil.PrincipalId].(string),
			event.RequestContext.Authorizer[authorizerutil.RoleName].(string),
		).String(),
	); err != nil {
		return nil, err
	}

	body, err := lambdautil.RenderHTML(credentialFactoryAuthorizeTemplate(), token)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil
}

func credentialFactoryFetchHandler(ctx context.Context, cfg *awscfg.Main, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	sess, err := awssessions.NewSession(awssessions.Config{})
	if err != nil {
		return nil, err
	}
	svc := iam.New(sess)

	// Requests to this endpoint are not authenticated or authorized by API
	// Gateway. Instead, we authorize requests by their presentation of a
	// valid bearer token. Validity is determined by finding a matching tag
	// on the CredentialFactory IAM user.
	token, ok := event.QueryStringParameters["token"]
	if !ok {
		return lambdautil.ErrorResponseJSON(
			http.StatusForbidden,
			errors.New("query string parameter token is required"),
		)
	}
	tags, err := awsiam.ListUserTags(svc, users.CredentialFactory)
	if err != nil {
		return nil, err
	}

	// HERE BE DRAGONS!
	//
	// If execution reaches passes this point without proper authorization then
	// very privileged AWS credentials will leak to whomever made the request.
	tagValue, err := ParseTagValue(tags[TagKeyPrefix+token])
	if err != nil {
		return lambdautil.ErrorResponseJSON(
			http.StatusForbidden,
			errors.New("token not previously authorized"),
		)
	}
	if tagValue.Expired() {
		return lambdautil.ErrorResponseJSON(
			http.StatusForbidden,
			errors.New("token authorization expired"),
		)
	}

	// Tokens are one-time use.
	if err := awsiam.UntagUser(
		svc,
		users.CredentialFactory,
		TagKeyPrefix+token,
	); err != nil {
		return nil, err
	}

	credentials, err := getCredentials(sess, tagValue.PrincipalId, tagValue.RoleName)
	if err != nil {
		return lambdautil.ErrorResponseJSON(http.StatusInternalServerError, err)
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
}

func getCredentials(sess *session.Session, principalId, roleName string) (*sts.Credentials, error) {
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

	time.Sleep(5e9) // I really wish I didn't have to do this

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
		roles.Arn(aws.StringValue(callerIdentity.Account), roleName),
		principalId,
		43200,
	)
	if err != nil {
		return nil, err
	}

	return assumedRole.Credentials, nil
}

func init() {
	handlers["/credential-factory"] = credentialFactoryHandler
	handlers["/credential-factory/authorize"] = credentialFactoryAuthorizeHandler
	handlers["/credential-factory/fetch"] = credentialFactoryFetchHandler
}
