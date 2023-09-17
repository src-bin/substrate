package audit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/ui"
)

func Main(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayProxyRequest,
) (*events.APIGatewayProxyResponse, error) {

	body, err := lambdautil.EventBody(event)
	if err != nil {
		return nil, err
	}

	e := telemetry.NewEmptyEvent()
	if err := json.Unmarshal([]byte(body), e); err != nil {
		return nil, err
	}
	ui.PrintfWithCaller("relaying telemetry %s", jsonutil.MustOneLineString(e))
	if err := e.Post(ctx); err != nil {
		return nil, err
	}
	if err := e.Wait(ctx); err != nil {
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: http.StatusAccepted,
	}, nil
}

func Main2(
	ctx context.Context,
	cfg *awscfg.Config,
	oc *oauthoidc.Client,
	event *events.APIGatewayV2HTTPRequest,
) (*events.APIGatewayV2HTTPResponse, error) {

	body, err := lambdautil.EventBody2(event)
	if err != nil {
		return nil, err
	}

	e := telemetry.NewEmptyEvent()
	if err := json.Unmarshal([]byte(body), e); err != nil {
		return nil, err
	}
	ui.PrintfWithCaller("relaying telemetry %s", jsonutil.MustOneLineString(e))
	if err := e.Post(ctx); err != nil {
		return nil, err
	}
	if err := e.Wait(ctx); err != nil {
		return nil, err
	}

	return &events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusAccepted,
	}, nil
}
