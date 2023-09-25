package awsapigatewayv2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/ui"
)

const authorizerTTL = 60 // 0 // the UI shows 300 if this is 0

func EnsureAuthorizer(
	ctx context.Context,
	cfg *awscfg.Config,
	apiId, name, roleARN, functionARN string,
) (authorizerId string, err error) {
	ui.Spinf("finding or creating the %s API Gateway v2 authorizer", name)
	client := cfg.APIGatewayV2()

	authorizerURI := fmt.Sprintf(
		"arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/%s/invocations",
		cfg.Region(),
		functionARN,
	)
	identitySource := []string{"$request.header.Cookie"} // OK to use Cookie headers with CloudFront redirecting requests without cookies

	var out *apigatewayv2.CreateAuthorizerOutput
	out, err = client.CreateAuthorizer(ctx, &apigatewayv2.CreateAuthorizerInput{
		ApiId:                          aws.String(apiId),
		AuthorizerCredentialsArn:       aws.String(roleARN),
		AuthorizerPayloadFormatVersion: aws.String("2.0"),
		AuthorizerType:                 types.AuthorizerTypeRequest,
		AuthorizerUri:                  aws.String(authorizerURI),
		AuthorizerResultTtlInSeconds:   authorizerTTL,
		IdentitySource:                 identitySource,
		Name:                           aws.String(naming.Substrate),
	})
	if err == nil {
		authorizerId = aws.ToString(out.AuthorizerId)
	} else if awsutil.ErrorCodeIs(err, BadRequestException) {

		var authorizer *types.Authorizer
		if authorizer, err = getAuthorizerByName(ctx, cfg, apiId, name); err != nil {
			ui.StopErr(err)
			return
		}
		authorizerId = aws.ToString(authorizer.AuthorizerId)

		_, err = client.UpdateAuthorizer(ctx, &apigatewayv2.UpdateAuthorizerInput{
			ApiId:                          aws.String(apiId),
			AuthorizerCredentialsArn:       aws.String(roleARN),
			AuthorizerId:                   authorizer.AuthorizerId,
			AuthorizerPayloadFormatVersion: aws.String("2.0"),
			AuthorizerResultTtlInSeconds:   authorizerTTL,
			AuthorizerType:                 types.AuthorizerTypeRequest,
			AuthorizerUri:                  aws.String(authorizerURI),
			IdentitySource:                 identitySource,
			Name:                           aws.String(naming.Substrate),
		})

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
	target := fmt.Sprintf("integrations/%s", aws.ToString(integration.IntegrationId))
	if err = EnsureRoute(ctx, cfg, apiId, []string{"GET", "POST"}, "/login", "", target); err != nil { // no authorizer for /login
		ui.StopErr(err)
		return
	}
	err = UpdateRoute(ctx, cfg, apiId, Default, authorizerId, target) // authorizer for every other route
	if err != nil {
		ui.StopErr(err)
		return
	}

	ui.Stopf("authorizer %s", authorizerId)
	return
}

func getAuthorizerByName(ctx context.Context, cfg *awscfg.Config, apiId, name string) (*types.Authorizer, error) {
	authorizers, err := getAuthorizers(ctx, cfg, apiId)
	if err != nil {
		return nil, err
	}
	for _, authorizer := range authorizers {
		if aws.ToString(authorizer.Name) == name {
			return &authorizer, nil
		}
	}
	return nil, NotFound{name, "authorizer"}
}

func getAuthorizers(ctx context.Context, cfg *awscfg.Config, apiId string) (authorizers []types.Authorizer, err error) {
	client := cfg.APIGatewayV2()
	var nextToken *string
	for {
		out, err := client.GetAuthorizers(ctx, &apigatewayv2.GetAuthorizersInput{
			ApiId:     aws.String(apiId),
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, authorizer := range out.Items {
			authorizers = append(authorizers, authorizer)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
