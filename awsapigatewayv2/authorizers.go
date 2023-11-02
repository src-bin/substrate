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
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

const authorizerTTL = 0 // HERE BE DRAGONS! Don't run any TTL except 0 unless the identity source is truly user-scoped

func EnsureAuthorizer(
	ctx context.Context,
	cfg *awscfg.Config,
	apiId, name, roleARN, functionARN string,
) (authorizerId string, err error) {
	ui.Spinf("finding or creating the %s API Gateway v2 authorizer", name)
	client := cfg.APIGatewayV2()
	region := cfg.Region()

	authorizerURI := fmt.Sprintf(
		"arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/%s/invocations",
		region,
		functionARN,
	)
	//identitySource := []string{"$request.header.Host"} // HERE BE DRAGONS! Don't run any TTL except 0 with this identity source
	//identitySource := []string{"$request.header.Cookie"} // it'd be nice if this worked but for some reason it causes lots of 403s
	identitySource := []string{"$context.requestId"} // make the authorizer run on every request

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

	// Use this authorizer for every route except GET /credential-factory/fetch
	// and {GET,POST} /login. Yes, this is a leak in the abstraction to pretend
	// this is a generalized AWS API Gateway management client but, hey, it's
	// not and that doesn't matter to Substrate (yet).
	var integration *types.Integration
	if integration, err = getIntegrationByFunctionARN(ctx, cfg, apiId, functionARN); err != nil {
		ui.StopErr(err)
		return
	}
	target := fmt.Sprintf("integrations/%s", aws.ToString(integration.IntegrationId))
	if err = EnsureRoute(ctx, cfg, apiId, []string{"GET"}, "/credential-factory/fetch", "", target); err != nil {
		ui.StopErr(err)
		return
	}
	if err = EnsureRoute(ctx, cfg, apiId, []string{"GET", "POST"}, "/login", "", target); err != nil {
		ui.StopErr(err)
		return
	}
	if err = UpdateRoute(ctx, cfg, apiId, Default, authorizerId, target); err != nil {
		ui.StopErr(err)
		return
	}

	// Tag the API with its authorizer ID so we can get to it in Terraform,
	// which doesn't include a data source for authorizers for some reason.
	if _, err = client.TagResource(ctx, &apigatewayv2.TagResourceInput{
		ResourceArn: aws.String(fmt.Sprintf("arn:aws:apigateway:%s::/apis/%s", region, apiId)),
		Tags:        tagging.Map{"AuthorizerId": authorizerId},
	}); err != nil {
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
