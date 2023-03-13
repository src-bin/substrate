package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/users"
)

//go:generate go run ../../tools/template/main.go -name credentialFactoryAuthorizeTemplate -package main credential-factory-authorize.html
//go:generate go run ../../tools/template/main.go -name credentialFactoryTemplate -package main credential-factory.html

// TODO use session policies <https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/credentials/stscreds#AssumeRoleOptions> to constrain where these credentials can go by e.g. lists of domains, environments, and/or qualities they're allowed to assume roles into (though we also need to account for Instance Factory instances)

const (
	GCExpiredTagsLimit         = 30 // higher than GCExpiredTagsSyncThreshold so we'll make progress even if it's growing
	GCExpiredTagsSyncThreshold = 25 // the default limit is 50

	MinTokenLength = 40

	TagKeyPrefix   = "CredentialFactory:" // duplicated in tools/garbage-credential-factory-tags/main.go
	TagValueFormat = "%s %s expiry %s"    // duplicated in tools/garbage-credential-factory-tags/main.go
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
	var expiry string
	v := &TagValue{}
	_, err := fmt.Sscanf(raw, TagValueFormat, &v.PrincipalId, &v.RoleName, &expiry)
	if err != nil {
		return nil, fmt.Errorf(`"%s" %w`, raw, err)
	}
	v.Expiry, err = time.Parse(time.RFC3339, expiry)
	return v, err
}

func (v *TagValue) Expired() bool {
	return time.Now().After(v.Expiry)
}

func (v *TagValue) String() string {
	return fmt.Sprintf( // imitated in tools/garbage-credential-factory-tags/main.go
		TagValueFormat,
		v.PrincipalId,
		v.RoleName,
		v.Expiry.Format(time.RFC3339),
	)
}

func credentialFactoryHandler(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayProxyRequest,
) (*events.APIGatewayProxyResponse, error) {

	creds, err := awsiam.AllDayCredentials(
		ctx,
		cfg,
		event.RequestContext.Authorizer[authorizerutil.RoleName].(string),
	)
	if err != nil {
		return lambdautil.ErrorResponse(err)
	}

	body, err := lambdautil.RenderHTML(credentialFactoryTemplate(), creds)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html"},
		StatusCode: http.StatusOK,
	}, nil
}

func credentialFactoryAuthorizeHandler(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayProxyRequest,
) (*events.APIGatewayProxyResponse, error) {

	// Garbage-collect expired tags synchronously, before we try to tag the
	// user for this run, if we're close to the limit.
	tags, err := awsiam.ListUserTags(ctx, cfg, users.CredentialFactory)
	if err != nil {
		return nil, err
	}
	if len(tags) > GCExpiredTagsSyncThreshold {
		gcExpiredTags(ctx, cfg, tags)
	}

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
		tagging.Map{TagKeyPrefix + token: NewTagValue(
			event.RequestContext.Authorizer[authorizerutil.PrincipalId].(string),
			event.RequestContext.Authorizer[authorizerutil.RoleName].(string),
		).String()},
	); err != nil {
		return nil, err
	}
	log.Printf("authorized a token exchange for %s", event.RequestContext.Authorizer[authorizerutil.PrincipalId].(string))

	// Garbage collect expired tags asynchronously if we're not very close to
	// the limit. Even though we're sending this into a goroutine, start it
	// definitively after the much more important awsiam.TagUser call so as to
	// avoid the ConcurrentModification error that can ruin things if we're
	// tagging and untagging at the same time.
	if len(tags) <= GCExpiredTagsSyncThreshold {
		go gcExpiredTags(context.Background(), cfg, tags)
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

func credentialFactoryFetchHandler(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayProxyRequest,
) (*events.APIGatewayProxyResponse, error) {
	ctx, _ = context.WithDeadline(ctx, time.Now().Add(28*time.Second)) // API Gateway's maximum wait time is 29 seconds

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
	//log.Print(jsonutil.MustOneLineString(tags))

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
	//log.Printf("found tag key %s with value %s", TagKeyPrefix+token, tagValue)
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
		[]string{TagKeyPrefix + token},
	); err != nil {
		return nil, err
	}
	//log.Printf("deleted tag key %s with value %s", TagKeyPrefix+token, tagValue)

	creds, err := awsiam.AllDayCredentials(

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
	log.Printf("exchanged token %s %s for access key %s", TagKeyPrefix+token, tagValue, creds.AccessKeyID)

	body, err := json.MarshalIndent(creds, "", "\t")
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayProxyResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: http.StatusOK,
	}, nil
}

func gcExpiredTags(ctx context.Context, cfg *awscfg.Config, tags tagging.Map) {
	keys := make([]string, 0, GCExpiredTagsLimit)
	for key, raw := range tags {
		if strings.HasPrefix(key, TagKeyPrefix) || strings.HasPrefix(key, "substrate-credential-factory:") /* old format */ {
			value, err := ParseTagValue(raw)
			if err == nil {
				if value.Expired() {
					keys = append(keys, key) // append instead of using [i] in case there are fewer than GCExpiredTagsLimit
				}
			} else {
				keys = append(keys, key)
				if !errors.Is(err, io.EOF) {
					log.Print(err)
				}
			}
		}
		if len(keys) >= GCExpiredTagsLimit {
			break
		}
	}
	//log.Print(keys)
	if err := awsiam.UntagUser(ctx, cfg, users.CredentialFactory, keys); err != nil {
		log.Print(err)
	}
}

func init() {
	handlers["/credential-factory"] = credentialFactoryHandler
	handlers["/credential-factory/authorize"] = credentialFactoryAuthorizeHandler
	handlers["/credential-factory/fetch"] = credentialFactoryFetchHandler
}
