package audit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/lambdautil"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/telemetry"
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
