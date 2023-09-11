package awsapigatewayv2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscloudwatch"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func EnsureAPI(
	ctx context.Context,
	cfg *awscfg.Config,
	name, roleARN, functionARN string,
) (apiId string, err error) {
	ui.Spinf("finding or creating the %s HTTP gateway (with API Gateway v2)", name)
	client := cfg.APIGatewayV2()

	if err = awscloudwatch.EnsureLogGroup(ctx, cfg, fmt.Sprintf("/aws/apigatewayv2/%s", name), 7); err != nil {
		ui.StopErr(err)
		return
	}

	var api *types.Api
	api, err = getAPIByName(ctx, cfg, name)
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
			apiId = aws.ToString(out.ApiId)
		}
	} else {
		var out *apigatewayv2.UpdateApiOutput
		if out, err = client.UpdateApi(ctx, &apigatewayv2.UpdateApiInput{
			ApiId:          api.ApiId,
			CredentialsArn: aws.String(roleARN),
			Name:           aws.String(name),
			Target:         aws.String(functionARN),
		}); err == nil {
			apiId = aws.ToString(out.ApiId)
		}
	}
	if err != nil {
		ui.StopErr(err)
		return
	}

	var integration *types.Integration
	if integration, err = getIntegrationByFunctionARN(ctx, cfg, apiId, functionARN); err != nil {
		ui.StopErr(err)
		return
	}
	if _, err = client.UpdateIntegration(ctx, &apigatewayv2.UpdateIntegrationInput{
		ApiId:         aws.String(apiId),
		IntegrationId: integration.IntegrationId,

		// I wish we could do HSTS and respond 302 Found to redirect to login
		// right here in API Gateway but, of course, they sabotaged themselves
		// and made this impossible. Being precise, these ResponseParameters
		// settings _work_ but they aren't consulted at all when an authorizer
		// is responding and so they have no utility in a browser-based
		// authentication and authorization flow.
		/*
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
		*/

	}); err != nil {
		ui.StopErr(err)
		return
	}

	if _, err = client.UpdateStage(ctx, &apigatewayv2.UpdateStageInput{
		AccessLogSettings: &types.AccessLogSettings{
			DestinationArn: aws.String(fmt.Sprintf(
				"arn:aws:logs:%s:%s:log-group:/aws/apigatewayv2/%s",
				cfg.Region(),
				cfg.MustAccountId(ctx),
				name,
			)),
			Format: aws.String(`$context.identity.sourceIp - - [$context.requestTime] "$context.httpMethod $context.routeKey $context.protocol" $context.status $context.responseLength $context.requestId`), // Apache common log format
		},
		ApiId:      aws.String(apiId),
		AutoDeploy: true,
		StageName:  aws.String("$default"),
	}); err != nil {
		ui.StopErr(err)
		return
	}

	ui.Stopf("API %s", apiId)
	return
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
