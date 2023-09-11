package awsapigatewayv2

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

const Default = "$default"

func EnsureRoute(
	ctx context.Context,
	cfg *awscfg.Config,
	apiId string,
	methods []string,
	path string,
	authorizerId, functionARN string,
) error {
	client := cfg.APIGatewayV2()
	for _, method := range methods {
		routeKey := fmt.Sprintf("%s %s", strings.ToUpper(method), path)
		in := &apigatewayv2.CreateRouteInput{
			ApiId:    aws.String(apiId),
			RouteKey: aws.String(routeKey),
			Target:   aws.String(functionARN),
		}
		if authorizerId == "" {
			in.AuthorizationType = types.AuthorizationTypeNone
		} else {
			in.AuthorizationType = types.AuthorizationTypeCustom
			in.AuthorizerId = aws.String(authorizerId)
		}
		_, err := client.CreateRoute(ctx, in)
		if awsutil.ErrorCodeIs(err, ConflictException) {
			var route *types.Route
			if route, err = getRouteByKey(ctx, cfg, apiId, routeKey); err != nil {
				return err
			}
			in := &apigatewayv2.UpdateRouteInput{
				ApiId:   aws.String(apiId),
				RouteId: route.RouteId,
				Target:  aws.String(functionARN),
			}
			if authorizerId == "" {
				in.AuthorizationType = types.AuthorizationTypeNone
			} else {
				in.AuthorizationType = types.AuthorizationTypeCustom
				in.AuthorizerId = aws.String(authorizerId)
			}
			_, err = client.UpdateRoute(ctx, in)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateRoute(
	ctx context.Context,
	cfg *awscfg.Config,
	apiId, routeKey string,
	authorizerId, functionARN string,
) error {
	route, err := getRouteByKey(ctx, cfg, apiId, routeKey)
	if err != nil {
		return err
	}
	in := &apigatewayv2.UpdateRouteInput{
		ApiId:   aws.String(apiId),
		RouteId: route.RouteId,
		Target:  aws.String(functionARN),
	}
	if authorizerId == "" {
		in.AuthorizationType = types.AuthorizationTypeNone
	} else {
		in.AuthorizationType = types.AuthorizationTypeCustom
		in.AuthorizerId = aws.String(authorizerId)
	}
	_, err = cfg.APIGatewayV2().UpdateRoute(ctx, in)
	return err
}

func getRouteByKey(ctx context.Context, cfg *awscfg.Config, apiId, key string) (*types.Route, error) {
	routes, err := getRoutes(ctx, cfg, apiId)
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		if aws.ToString(route.RouteKey) == key {
			return &route, nil
		}
	}
	return nil, NotFound{key, "route"}
}

func getRoutes(ctx context.Context, cfg *awscfg.Config, apiId string) (routes []types.Route, err error) {
	client := cfg.APIGatewayV2()
	var nextToken *string
	for {
		out, err := client.GetRoutes(ctx, &apigatewayv2.GetRoutesInput{
			ApiId:     aws.String(apiId),
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, route := range out.Items {
			routes = append(routes, route)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
