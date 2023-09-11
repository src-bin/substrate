package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
)

type Mux struct {
	Authorizer func(
		context.Context,
		*events.APIGatewayV2CustomAuthorizerV2Request,
	) (*events.APIGatewayV2CustomAuthorizerIAMPolicyResponse, error)
	Handler func(
		context.Context,
		*events.APIGatewayV2HTTPRequest,
	) (*events.APIGatewayV2HTTPResponse, error)
}

func (mux *Mux) Invoke(ctx context.Context, payload []byte) ([]byte, error) {

	// To avoid doing two passes of JSON decoding, we decode once into a
	// "superevent" that has every field from both
	// APIGatewayV2AuthorizerV2Request and APIGatewayV2HTTPRequest. After
	// decoding, we can decide based on the fields that are unique which
	// one to construct and which function to call.
	var superevent struct {
		Body                  string                                `json:"body"`
		Cookies               []string                              `json:"cookies"`
		Headers               map[string]string                     `json:"headers"`
		IdentitySource        []string                              `json:"identitySource"`
		IsBase64Encoded       bool                                  `json:"isBase64Encoded"`
		PathParameters        map[string]string                     `json:"pathParameters"`
		QueryStringParameters map[string]string                     `json:"queryStringParameters"`
		RawPath               string                                `json:"rawPath"`
		RawQueryString        string                                `json:"rawQueryString"`
		RequestContext        events.APIGatewayV2HTTPRequestContext `json:"requestContext"`
		RouteArn              string                                `json:"routeArn"`
		RouteKey              string                                `json:"routeKey"`
		StageVariables        map[string]string                     `json:"stageVariables"`
		Type                  string                                `json:"type"`
		Version               string                                `json:"version"`
	}
	if err := json.Unmarshal(payload, &superevent); err != nil {
		log.Print(err)
		return nil, err
	}

	var (
		response interface{}
		err      error
	)
	if superevent.RouteArn != "" {
		response, err = mux.Authorizer(ctx, &events.APIGatewayV2CustomAuthorizerV2Request{
			Version:               superevent.Version,
			Type:                  superevent.Type,
			RouteArn:              superevent.RouteArn,
			IdentitySource:        superevent.IdentitySource,
			RouteKey:              superevent.RouteKey,
			RawPath:               superevent.RawPath,
			RawQueryString:        superevent.RawQueryString,
			Cookies:               superevent.Cookies,
			Headers:               superevent.Headers,
			QueryStringParameters: superevent.QueryStringParameters,
			RequestContext:        superevent.RequestContext,
			PathParameters:        superevent.PathParameters,
			StageVariables:        superevent.StageVariables,
		})
	} else {
		response, err = mux.Handler(ctx, &events.APIGatewayV2HTTPRequest{
			Version:               superevent.Version,
			RouteKey:              superevent.RouteKey,
			RawPath:               superevent.RawPath,
			RawQueryString:        superevent.RawQueryString,
			Cookies:               superevent.Cookies,
			Headers:               superevent.Headers,
			QueryStringParameters: superevent.QueryStringParameters,
			PathParameters:        superevent.PathParameters,
			RequestContext:        superevent.RequestContext,
			StageVariables:        superevent.StageVariables,
			Body:                  superevent.Body,
			IsBase64Encoded:       superevent.IsBase64Encoded,
		})
	}
	if err != nil {
		log.Print(err)
		return nil, err
	}

	return json.MarshalIndent(response, "", "\t")
}
