package awsapigatewayv2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

func EnsureIntegration(
	ctx context.Context,
	cfg *awscfg.Config,
	apiId, roleARN, functionARN string,
	// TODO PassthroughBehavior, {Request,Response}Parameters, etc.
) (integrationId string, err error) {
	client := cfg.APIGatewayV2()

	var out *apigatewayv2.CreateIntegrationOutput
	out, err = client.CreateIntegration(ctx, &apigatewayv2.CreateIntegrationInput{
		ApiId:                aws.String(apiId),
		CredentialsArn:       aws.String(roleARN),
		PayloadFormatVersion: aws.String("2.0"),
		IntegrationMethod:    aws.String("POST"),
		IntegrationType:      types.IntegrationTypeAws,
		IntegrationUri:       aws.String(functionARN),
	})
	if err == nil {
		integrationId = aws.ToString(out.IntegrationId)
	} else if awsutil.ErrorCodeIs(err, ConflictException) {

		var integration *types.Integration
		if integration, err = getIntegrationByFunctionARN(ctx, cfg, apiId, functionARN); err != nil {
			return
		}
		integrationId = aws.ToString(integration.IntegrationId)

		_, err = client.UpdateIntegration(ctx, &apigatewayv2.UpdateIntegrationInput{
			ApiId:                aws.String(apiId),
			CredentialsArn:       aws.String(roleARN),
			IntegrationId:        aws.String(integrationId),
			PayloadFormatVersion: aws.String("2.0"),
			IntegrationMethod:    aws.String("POST"),
			IntegrationType:      types.IntegrationTypeAws,
			IntegrationUri:       aws.String(functionARN),
		})

	}

	return
}

func getIntegrationByFunctionARN(ctx context.Context, cfg *awscfg.Config, apiId, functionARN string) (*types.Integration, error) {
	integrations, err := getIntegrations(ctx, cfg, apiId)
	if err != nil {
		return nil, err
	}
	for _, integration := range integrations {
		if aws.ToString(integration.IntegrationUri) == functionARN {
			return &integration, nil
		}
	}
	return nil, NotFound{functionARN, "integration"}
}

func getIntegrations(ctx context.Context, cfg *awscfg.Config, apiId string) (integrations []types.Integration, err error) {
	client := cfg.APIGatewayV2()
	var nextToken *string
	for {
		out, err := client.GetIntegrations(ctx, &apigatewayv2.GetIntegrationsInput{
			ApiId:     aws.String(apiId),
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, integration := range out.Items {
			integrations = append(integrations, integration)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
