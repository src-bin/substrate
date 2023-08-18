package awsapigatewayv2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

type API = types.Api

func EnsureAPI(ctx context.Context, cfg *awscfg.Config, name, roleARN, functionARN string) (apiId string, err error) {
	ui.Spinf("finding or creating the %s API (with API Gateway v2)", name)
	var api *API
	api, err = getAPIByName(ctx, cfg, name)
	if _, ok := err.(NotFound); ok {
		var out *apigatewayv2.CreateApiOutput
		out, err = cfg.APIGatewayV2().CreateApi(ctx, &apigatewayv2.CreateApiInput{
			CredentialsArn: aws.String(roleARN),
			Name:           aws.String(name),
			ProtocolType:   types.ProtocolTypeHttp,
			Tags: tagging.Map{
				tagging.Manager:          tagging.Substrate,
				tagging.SubstrateVersion: version.Version,
			},
			Target: aws.String(functionARN),
		})
		if err == nil {
			apiId = aws.ToString(out.ApiId)
		}
	} else {
		var out *apigatewayv2.UpdateApiOutput
		out, err = cfg.APIGatewayV2().UpdateApi(ctx, &apigatewayv2.UpdateApiInput{
			ApiId:          api.ApiId,
			CredentialsArn: aws.String(roleARN),
			Name:           aws.String(name),
			Target:         aws.String(functionARN),
		})
		apiId = aws.ToString(out.ApiId)
	}
	ui.StopErr(err)
	return
}

type NotFound string

func (err NotFound) Error() string {
	return fmt.Sprintf("API not found: %s", string(err))
}

func getAPIByName(ctx context.Context, cfg *awscfg.Config, name string) (*API, error) {
	apis, err := getAPIs(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, api := range apis {
		if aws.ToString(api.Name) == name {
			return &api, nil
		}
	}
	return nil, NotFound(name)
}

func getAPIs(ctx context.Context, cfg *awscfg.Config) (apis []API, err error) {
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
