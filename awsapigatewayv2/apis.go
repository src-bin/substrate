package awsapigatewayv2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/src-bin/substrate/awsacm"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscloudwatch"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

type API struct {
	Endpoint, Id string
}

func EnsureAPI(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	dnsDomainName, zoneId string,
	roleARN, functionARN string,
) (*API, error) {
	ui.Spinf("finding or creating the %s API Gateway v2", name)
	client := cfg.APIGatewayV2()

	if err := awscloudwatch.EnsureLogGroup(ctx, cfg, fmt.Sprintf("/aws/apigatewayv2/%s", name), 7); err != nil {
		return nil, ui.StopErr(err)
	}

	var api *API
	existingAPI, err := getAPIByName(ctx, cfg, name)
	if _, ok := err.(NotFound); ok {
		var out *apigatewayv2.CreateApiOutput
		if out, err = client.CreateApi(ctx, &apigatewayv2.CreateApiInput{
			CredentialsArn: aws.String(roleARN),
			Name:           aws.String(name),
			ProtocolType:   types.ProtocolTypeHttp,
			Tags: tagging.Map{
				tagging.Manager:          tagging.Substrate,
				tagging.SubstrateVersion: version.Version,
			},
			Target: aws.String(functionARN),
		}); err == nil {
			api = &API{
				Endpoint: aws.ToString(out.ApiEndpoint),
				Id:       aws.ToString(out.ApiId),
			}
		}
	} else {
		var out *apigatewayv2.UpdateApiOutput
		if out, err = client.UpdateApi(ctx, &apigatewayv2.UpdateApiInput{
			ApiId:          existingAPI.ApiId,
			CredentialsArn: aws.String(roleARN),
			Name:           aws.String(name),
			Target:         aws.String(functionARN),
		}); err == nil {
			api = &API{
				Endpoint: aws.ToString(out.ApiEndpoint),
				Id:       aws.ToString(out.ApiId),
			}
		}
	}
	if err != nil {
		return nil, ui.StopErr(err)
	}

	/*
		var integration *types.Integration
		if integration, err = getIntegrationByFunctionARN(ctx, cfg, api.Id, functionARN); err != nil {
			return nil, ui.StopErr(err)
		}
		if _, err = client.UpdateIntegration(ctx, &apigatewayv2.UpdateIntegrationInput{
			ApiId:         aws.String(api.Id),
			IntegrationId: integration.IntegrationId,

			// I wish we could do HSTS and respond 302 Found to redirect to login
			// right here in API Gateway but, of course, they sabotaged themselves
			// and made this impossible. Being precise, these ResponseParameters
			// settings _work_ but they aren't consulted at all when an authorizer
			// is responding and so they have no utility in a browser-based
			// authentication and authorization flow.
			ResponseParameters: map[string]map[string]string{
				"200": map[string]string{"append:header.Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload"},
				"302": map[string]string{"append:header.Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload"},
				"401": map[string]string{
					"append:header.Location":                  "$context.authorizer.Location",
					"append:header.Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
					"overwrite:statuscode":                    "302",
				},
				"403": map[string]string{
					"append:header.Location":                  "$context.authorizer.Location",
					"append:header.Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
					"overwrite:statuscode":                    "302",
				},
				"404": map[string]string{"append:header.Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload"},
				"500": map[string]string{"append:header.Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload"},
			},
		}); err != nil {
			return nil, ui.StopErr(err)
		}
	*/

	if _, err = client.UpdateStage(ctx, &apigatewayv2.UpdateStageInput{
		AccessLogSettings: &types.AccessLogSettings{
			DestinationArn: aws.String(fmt.Sprintf(
				"arn:aws:logs:%s:%s:log-group:/aws/apigatewayv2/%s",
				cfg.Region(),
				cfg.MustAccountId(ctx),
				name,
			)),
			//Format: aws.String(`$context.identity.sourceIp - - [$context.requestTime] "$context.httpMethod $context.routeKey $context.protocol" $context.status $context.responseLength $context.requestId`), // Apache common log format
			Format: aws.String(jsonutil.MustOneLineString(map[string]string{
				"accountId":                     "$context.accountId",
				"apiId":                         "$context.apiId",
				"authorizer.error":              "$context.authorizer.error",
				"authorizer.principalId":        "$context.authorizer.principalId",
				"awsEndpointRequestId":          "$context.awsEndpointRequestId",
				"awsEndpointRequestId2":         "$context.awsEndpointRequestId2",
				"customDomain.basePathMatched":  "$context.customDomain.basePathMatched",
				"dataProcessed":                 "$context.dataProcessed",
				"domainName":                    "$context.domainName",
				"domainPrefix":                  "$context.domainPrefix",
				"error.message":                 "$context.error.message",
				"error.responseType":            "$context.error.responseType",
				"httpMethod":                    "$context.httpMethod",
				"integration.error":             "$context.integration.error",
				"integrationErrorMessage":       "$context.integrationErrorMessage",
				"integration.integrationStatus": "$context.integration.integrationStatus",
				"integration.latency":           "$context.integration.latency",
				"integrationLatency":            "$context.integrationLatency",
				"integration.requestId":         "$context.integration.requestId",
				"integration.status":            "$context.integration.status",
				"integrationStatus":             "$context.integrationStatus",
				"path":                          "$context.path",
				"protocol":                      "$context.protocol",
				"requestId":                     "$context.requestId",
				"requestTime":                   "$context.requestTime",
				"requestTimeEpoch":              "$context.requestTimeEpoch",
				"responseLatency":               "$context.responseLatency",
				"responseLength":                "$context.responseLength",
				"routeKey":                      "$context.routeKey",
				"stage":                         "$context.stage",
				"status":                        "$context.status",
			})), // almost everything per <https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-logging-variables.html>
		},
		ApiId:      aws.String(api.Id),
		AutoDeploy: true,
		StageName:  aws.String("$default"),
	}); err != nil {
		return nil, ui.StopErr(err)
	}

	cert, err := awsacm.EnsureCertificate(ctx, cfg, dnsDomainName, []string{dnsDomainName}, zoneId)
	if err != nil {
		return nil, ui.StopErr(err)
	}

	ui.Stopf("API %s", api.Id)
	return api, nil
}

func getAPIByName(ctx context.Context, cfg *awscfg.Config, name string) (*types.Api, error) {
	apis, err := getAPIs(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, api := range apis {
		if aws.ToString(api.Name) == name {
			return &api, nil
		}
	}
	return nil, NotFound{name, "API"}
}

func getAPIs(ctx context.Context, cfg *awscfg.Config) (apis []types.Api, err error) {
	client := cfg.APIGatewayV2()
	var nextToken *string
	for {
		out, err := client.GetApis(ctx, &apigatewayv2.GetApisInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, api := range out.Items {
			apis = append(apis, api)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
