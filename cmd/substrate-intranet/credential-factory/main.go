package credentialfactory

import (
	"context"
	_ "embed"
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
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	GCExpiredTagsLimit         = 30 // higher than GCExpiredTagsSyncThreshold so we'll make progress even if it's growing
	GCExpiredTagsSyncThreshold = 25 // the default limit is 50

	MinTokenLength = 40

	TagKeyPrefix   = "CredentialFactory:" // duplicated in tools/garbage-credential-factory-tags/main.go
	TagValueFormat = "%s %s expiry %s"    // duplicated in tools/garbage-credential-factory-tags/main.go
)

func Main2(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	switch event.RawPath {
	case "/credential-factory":
		return index(ctx, cfg, oc, event)
	case "/credential-factory/authorize":
		return authorize(ctx, cfg, oc, event)
	case "/credential-factory/fetch":
		return fetch(ctx, cfg, oc, event)
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       fmt.Sprintf("%s not found\n", event.RawPath),
		Headers:    map[string]string{"Content-Type": "text/plain"},
		StatusCode: http.StatusNotFound,
	}, nil
}

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

func authorize(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {

	// Garbage-collect expired tags synchronously, before we try to tag the
	// user for this run, if we're close to the limit.
	tags, err := awsiam.ListUserTags(ctx, cfg, users.Substrate)
	if err != nil {
		return nil, err
	}
	if len(tags) > GCExpiredTagsSyncThreshold {
		gcExpiredTags(ctx, cfg, tags)
	}

	// Tag the Substrate IAM user using the bearer token as the key and the
	// session name as the value. We choose to use tags as our database here
	// because there won't be that many and it's free. We choose to use tags on
	// an IAM resource because they're global and Substrate's Intranet is
	// multi-region.
	token, ok := event.QueryStringParameters["token"]
	if !ok {
		return lambdautil.ErrorResponse2(errors.New("query string parameter token is required"))
	}
	if len(token) < MinTokenLength {
		return lambdautil.ErrorResponse2(fmt.Errorf("token must be at least %d characters long", MinTokenLength))
	}
	if err := awsiam.TagUser(
		ctx,
		cfg,
		users.Substrate,
		tagging.Map{TagKeyPrefix + token: NewTagValue(
			fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.PrincipalId]),
			fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.RoleName]),
		).String()},
	); err != nil {
		return nil, err
	}
	ui.PrintfWithCaller(
		"authorized a token exchange for %s",
		fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.PrincipalId]),
	)

	// Garbage collect expired tags asynchronously if we're not very close to
	// the limit. Even though we're sending this into a goroutine, start it
	// definitively after the much more important awsiam.TagUser call so as to
	// avoid the ConcurrentModification error that can ruin things if we're
	// tagging and untagging at the same time.
	if len(tags) <= GCExpiredTagsSyncThreshold {
		go gcExpiredTags(context.Background(), cfg, tags)
	}

	body, err := lambdautil.RenderHTML(htmlForAuthorize, token)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		StatusCode: http.StatusOK,
	}, nil
}

func fetch(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	ctx, _ = context.WithDeadline(ctx, time.Now().Add(28*time.Second)) // API Gateway's maximum wait time is 29 seconds

	// Requests to this endpoint are not authenticated or authorized by API
	// Gateway. Instead, we authorize requests by their presentation of a
	// valid bearer token. Validity is determined by finding a matching tag
	// on the Substrate IAM user.
	token, ok := event.QueryStringParameters["token"]
	if !ok {
		return lambdautil.ErrorResponseJSON2(
			http.StatusForbidden,
			errors.New("query string parameter token is required"),
		)
	}
	tags, err := awsiam.ListUserTags(ctx, cfg, users.Substrate)
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
		return lambdautil.ErrorResponseJSON2(
			http.StatusForbidden,
			errors.New("token not previously authorized"),
		)
	}
	//log.Printf("found tag key %s with value %s", TagKeyPrefix+token, tagValue)
	if tagValue.Expired() {
		return lambdautil.ErrorResponseJSON2(
			http.StatusForbidden,
			errors.New("token authorization expired"),
		)
	}

	// Tokens are one-time use.
	if err := awsiam.UntagUser(
		ctx,
		cfg,
		users.Substrate,
		[]string{TagKeyPrefix + token},
	); err != nil {
		return nil, err
	}
	//log.Printf("deleted tag key %s with value %s", TagKeyPrefix+token, tagValue)

	accountId, err := cfg.AccountId(ctx)
	if err != nil {
		return lambdautil.ErrorResponse2(err)
	}
	creds, err := awsiam.AllDayCredentials(

		// Since this API is unauthenticated, at least in the typical way, we
		// don't have the Username context set in the typical way, either. Fix
		// that up here so everything makes sense.
		context.WithValue(ctx, "Username", tagValue.PrincipalId),

		cfg,
		accountId,
		tagValue.RoleName,
	)
	if err != nil {
		return lambdautil.ErrorResponseJSON2(http.StatusInternalServerError, err)
	}
	ui.PrintfWithCaller("exchanged token %s %s for access key %s", TagKeyPrefix+token, tagValue, creds.AccessKeyID)

	body, err := json.MarshalIndent(creds, "", "\t")
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       string(body),
		Headers:    map[string]string{"Content-Type": "application/json; charset=utf-8"},
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
	if err := awsiam.UntagUser(ctx, cfg, users.Substrate, keys); err != nil {
		log.Print(err)
	}
}

func index(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {
	accountId, err := cfg.AccountId(ctx)
	if err != nil {
		return lambdautil.ErrorResponse2(err)
	}
	creds, err := awsiam.AllDayCredentials(
		ctx,
		cfg,
		accountId,
		fmt.Sprint(event.RequestContext.Authorizer.Lambda[authorizerutil.RoleName]),
	)
	if err != nil {
		return lambdautil.ErrorResponse2(err)
	}

	body, err := lambdautil.RenderHTML(html, creds)
	if err != nil {
		return nil, err
	}
	return &events.APIGatewayV2HTTPResponse{
		Body:       body,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		StatusCode: http.StatusOK,
	}, nil
}

//go:embed credential-factory.html
var html string

//go:embed authorize.html
var htmlForAuthorize string
