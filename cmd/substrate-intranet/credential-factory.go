package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/ui"
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

func credentialFactoryHandler(ctx context.Context, cfg *awscfg.Config, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	credentials, err := getCredentials(
		ctx,
		cfg,
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

func credentialFactoryAuthorizeHandler(ctx context.Context, cfg *awscfg.Config, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

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
		ctx,
		cfg,
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

func credentialFactoryFetchHandler(ctx context.Context, cfg *awscfg.Config, event *events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

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
	tags, err := awsiam.ListUserTags(ctx, cfg, users.CredentialFactory)
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
		ctx,
		cfg,
		users.CredentialFactory,
		TagKeyPrefix+token,
	); err != nil {
		return nil, err
	}

	credentials, err := getCredentials(

		// Since this API is unauthenticated, at least in the typical way, we
		// don't have the Username context set in the typical way, either. Fix
		// that up here so everything makes sense.
		context.WithValue(ctx, "Username", tagValue.PrincipalId),

		cfg,
		tagValue.RoleName,
	)
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

func getCredentials(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
) (credentials aws.Credentials, err error) {
	var accessKey *types.AccessKey
	for i := 0; i < CreateAccessKeyTriesTotal; i++ {
		accessKey, err = awsiam.CreateAccessKey(ctx, cfg, users.CredentialFactory)
		if awsutil.ErrorCodeIs(err, awsiam.LimitExceeded) {
			if i == CreateAccessKeyTriesBeforeDeleteAll {
				if err = awsiam.DeleteAllAccessKeys(ctx, cfg, users.CredentialFactory); err != nil {
					return
				}
			}
			continue
		}
		break
	}
	if err != nil {
		return
	}
	defer func() {
		if err := awsiam.DeleteAccessKey(
			ctx,
			cfg,
			users.CredentialFactory,
			aws.ToString(accessKey.AccessKeyId),
		); err != nil {
			ui.Print(err)
		}
	}()

	// Make a copy of the AWS SDK config that we're going to use to bounce
	// through user/CredentialFactory in order to get 12-hour credentials so
	// that we don't ruin cfg for whatever else we might want to do with it.
	cfg12h := cfg.Copy()

	callerIdentity, err := cfg12h.SetCredentials(ctx, aws.Credentials{
		AccessKeyID:     aws.ToString(accessKey.AccessKeyId),
		SecretAccessKey: aws.ToString(accessKey.SecretAccessKey),
	})
	if err != nil {
		ui.PrintWithCaller(err)
		return
	}
	cfg12h, err = cfg12h.AssumeRole(
		ctx,
		aws.ToString(callerIdentity.Account),
		roleName,
		12*time.Hour,
	)
	if err != nil {
		ui.PrintWithCaller(err)
		return
	}

	return cfg12h.Retrieve(ctx)
}

func init() {
	handlers["/credential-factory"] = credentialFactoryHandler
	handlers["/credential-factory/authorize"] = credentialFactoryAuthorizeHandler
	handlers["/credential-factory/fetch"] = credentialFactoryFetchHandler
}
