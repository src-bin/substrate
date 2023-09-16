package awscloudfront

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

const CachePolicyAlreadyExists = "CachePolicyAlreadyExists"

func ensureCachePolicy(ctx context.Context, cfg *awscfg.Config, name string) (*types.CachePolicy, error) {
	client := cfg.CloudFront()

	cachePolicyConfig := &types.CachePolicyConfig{
		DefaultTTL: aws.Int64(0),
		MaxTTL:     aws.Int64(1), // can't be 0 or we can't tell it to forward cookies to the origin
		MinTTL:     aws.Int64(0),
		Name:       aws.String(name),
		ParametersInCacheKeyAndForwardedToOrigin: &types.ParametersInCacheKeyAndForwardedToOrigin{
			CookiesConfig: &types.CachePolicyCookiesConfig{
				CookieBehavior: types.CachePolicyCookieBehaviorAll,
				/*
					Cookies: &types.CookieNames{
						Items:    []string{},
						Quantity: aws.Int32(0),
					},
				*/
			},
			EnableAcceptEncodingBrotli: aws.Bool(true),
			EnableAcceptEncodingGzip:   aws.Bool(true),
			HeadersConfig: &types.CachePolicyHeadersConfig{
				HeaderBehavior: types.CachePolicyHeaderBehaviorNone,
				/*
					Headers: &types.Headers{
						Items:    []string{"cookie"}, // "cookie" is not allowed because it gets special treatment
						Quantity: aws.Int32(1),
					},
				*/
			},
			QueryStringsConfig: &types.CachePolicyQueryStringsConfig{
				QueryStringBehavior: types.CachePolicyQueryStringBehaviorAll,
				/*
					QueryStrings: &types.QueryStringNames{
						Items:    []string{},
						Quantity: aws.Int32(0),
					},
				*/
			},
		},
	}

	out, err := client.CreateCachePolicy(ctx, &cloudfront.CreateCachePolicyInput{
		CachePolicyConfig: cachePolicyConfig,
	})
	if err == nil {
		return out.CachePolicy, nil
	} else if awsutil.ErrorCodeIs(err, CachePolicyAlreadyExists) {
		var etag, id *string
		{
			cachePolicy, err := getCachePolicyByName(ctx, cfg, name)
			if err != nil {
				return nil, err
			}
			out, err := client.GetCachePolicy(ctx, &cloudfront.GetCachePolicyInput{
				Id: cachePolicy.Id,
			})
			if err != nil {
				return nil, err
			}
			etag = out.ETag
			id = out.CachePolicy.Id
		}
		out, err := client.UpdateCachePolicy(ctx, &cloudfront.UpdateCachePolicyInput{
			CachePolicyConfig: cachePolicyConfig,
			Id:                id,
			IfMatch:           etag,
		})
		if err != nil {
			return nil, err
		}
		return out.CachePolicy, nil
	}
	return nil, err
}

func getCachePolicyByName(ctx context.Context, cfg *awscfg.Config, name string) (*types.CachePolicy, error) {
	cachePolicies, err := listCachePolicies(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, cachePolicy := range cachePolicies {
		if aws.ToString(cachePolicy.CachePolicyConfig.Name) == name {
			return cachePolicy, nil
		}
	}
	return nil, fmt.Errorf("cache policy %q not found", name)
}

func listCachePolicies(ctx context.Context, cfg *awscfg.Config) (cachePolicies []*types.CachePolicy, err error) {
	client := cfg.CloudFront()
	var nextMarker *string
	for {
		out, err := client.ListCachePolicies(ctx, &cloudfront.ListCachePoliciesInput{
			Marker: nextMarker,
			Type:   types.CachePolicyTypeCustom,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range out.CachePolicyList.Items {
			cachePolicies = append(cachePolicies, item.CachePolicy)
		}
		if nextMarker = out.CachePolicyList.NextMarker; nextMarker == nil {
			break
		}
	}
	return
}
