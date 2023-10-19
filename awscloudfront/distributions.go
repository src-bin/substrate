package awscloudfront

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/src-bin/substrate/awsacm"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const HostedZoneId = "Z2FDTNDATAQYW2" // CloudFront's zone ID in Route 53 for use when creating ALIAS records

type Distribution struct { // just the fields we want out of types.Distribution and types.DistributionSummary
	ARN, Comment, DomainName, Id string
}

func EnsureDistribution(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	subjectAlternativeNames []string, // all of these names...
	zoneId string, // ...must be in this zone
	functionEvents []EventType,
	functionCode string,
	originURL string,
) (*Distribution, error) {
	ui.Spinf("finding or creating the %s CloudFront distribution", name)
	client := cfg.CloudFront()

	cachePolicy, err := ensureCachePolicy(ctx, cfg, name)
	if err != nil {
		return nil, ui.StopErr(err)
	}
	var functionARN string
	if len(functionEvents) > 0 && len(functionCode) > 0 {
		functionARN, err = ensureFunction(ctx, cfg, name, functionCode)
		if err != nil {
			return nil, ui.StopErr(err)
		}
	}
	u, err := url.Parse(originURL)
	//ui.Debug(u)
	if err != nil {
		return nil, ui.StopErr(err)
	}
	distributionConfig := &types.DistributionConfig{
		Aliases: &types.Aliases{
			Items:    subjectAlternativeNames,
			Quantity: aws.Int32(int32(len(subjectAlternativeNames))),
		},
		CacheBehaviors: &types.CacheBehaviors{
			//Items:    []types.CacheBehavior{},
			Quantity: aws.Int32(0),
		},
		CallerReference: aws.String(time.Now().Format(time.RFC3339)),
		Comment:         aws.String(name),
		CustomErrorResponses: &types.CustomErrorResponses{
			//Items:    []types.CustomErrorResponse{},
			Quantity: aws.Int32(0),
		},
		DefaultCacheBehavior: &types.DefaultCacheBehavior{
			AllowedMethods: &types.AllowedMethods{
				CachedMethods: &types.CachedMethods{
					Items:    []types.Method{"GET", "HEAD", "OPTIONS"},
					Quantity: aws.Int32(3),
				},
				Items:    []types.Method{"GET", "HEAD", "POST", "PUT", "PATCH", "OPTIONS", "DELETE"},
				Quantity: aws.Int32(7),
			},
			CachePolicyId:          cachePolicy.Id,
			Compress:               aws.Bool(true),
			FieldLevelEncryptionId: aws.String(""),
			FunctionAssociations:   &types.FunctionAssociations{},
			LambdaFunctionAssociations: &types.LambdaFunctionAssociations{
				//Items:    []types.LambdaFunctionAssociation{},
				Quantity: aws.Int32(0),
			},
			SmoothStreaming:      aws.Bool(false),
			TargetOriginId:       aws.String(name),
			ViewerProtocolPolicy: types.ViewerProtocolPolicyRedirectToHttps,
		},
		DefaultRootObject: aws.String(""),
		Enabled:           aws.Bool(true),
		HttpVersion:       types.HttpVersionHttp2,
		IsIPV6Enabled:     aws.Bool(true),
		Logging: &types.LoggingConfig{
			Bucket:         aws.String(""),
			Enabled:        aws.Bool(false),
			IncludeCookies: aws.Bool(false),
			Prefix:         aws.String(""),
		},
		Origins: &types.Origins{
			Items: []types.Origin{{
				CustomHeaders: &types.CustomHeaders{
					//Items:    []types.OriginCustomHeader{},
					Quantity: aws.Int32(0),
				},
				CustomOriginConfig: &types.CustomOriginConfig{
					HTTPPort:               aws.Int32(80),
					HTTPSPort:              aws.Int32(443),
					OriginKeepaliveTimeout: aws.Int32(5), // default
					OriginProtocolPolicy:   types.OriginProtocolPolicyHttpsOnly,
					OriginReadTimeout:      aws.Int32(30), // default
					OriginSslProtocols: &types.OriginSslProtocols{
						Items:    []types.SslProtocol{types.SslProtocolTLSv12},
						Quantity: aws.Int32(1),
					},
				},
				DomainName: aws.String(u.Host),
				Id:         aws.String(name),
				OriginPath: aws.String(""),
			}},
			Quantity: aws.Int32(1),
		},
		PriceClass: types.PriceClassPriceClass100,
		Restrictions: &types.Restrictions{
			GeoRestriction: &types.GeoRestriction{
				Quantity:        aws.Int32(0),
				RestrictionType: types.GeoRestrictionTypeNone,
			},
		},
		ViewerCertificate: &types.ViewerCertificate{
			CloudFrontDefaultCertificate: aws.Bool(len(subjectAlternativeNames) == 0),
			MinimumProtocolVersion:       types.MinimumProtocolVersionTLSv122021,
			SSLSupportMethod:             types.SSLSupportMethodSniOnly,
		},
		WebACLId: aws.String(""),
	}
	for _, e := range functionEvents {
		distributionConfig.DefaultCacheBehavior.FunctionAssociations.Items = append(
			distributionConfig.DefaultCacheBehavior.FunctionAssociations.Items,
			types.FunctionAssociation{EventType: e, FunctionARN: aws.String(functionARN)},
		)
	}
	distributionConfig.DefaultCacheBehavior.FunctionAssociations.Quantity = aws.Int32(int32(
		len(distributionConfig.DefaultCacheBehavior.FunctionAssociations.Items),
	))
	if len(subjectAlternativeNames) > 0 {
		cert, err := awsacm.EnsureCertificate(
			ctx,
			cfg.Regional("us-east-1"),  // certificates from ACM for CloudFront must be in us-east-1
			subjectAlternativeNames[0], // CN, which doesn't actually matter so the first name's good enough
			subjectAlternativeNames,
			zoneId,
		)
		if err != nil {
			return nil, ui.StopErr(err)
		}
		distributionConfig.ViewerCertificate.ACMCertificateArn = cert.CertificateArn
	}
	//ui.Debug(distributionConfig)

	// If we can find an existing distribution with the same comment (which is
	// the closest thing we're going to get to a secondary unique index in
	// CloudFront) then update it according to the algorithm outlined in
	// <https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_UpdateDistribution.html>.
	if d, err := getDistributionByName(ctx, cfg, name); err == nil {
		out, err := client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{
			Id: d.Id,
		})
		if err != nil {
			return nil, ui.StopErr(err)
		}
		distributionConfig.CallerReference = out.DistributionConfig.CallerReference
		if _, err := client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
			DistributionConfig: distributionConfig,
			Id:                 d.Id,
			IfMatch:            out.ETag,
		}); err != nil {
			return nil, ui.StopErr(err)
		}
		if err := changeResourceRecordSets(ctx, cfg, subjectAlternativeNames, zoneId, d.DomainName); err != nil {
			return nil, ui.StopErr(err)
		}
		ui.Stop("ok")
		return &Distribution{
			ARN:        aws.ToString(d.ARN),
			Comment:    aws.ToString(d.Comment),
			DomainName: aws.ToString(d.DomainName),
			Id:         aws.ToString(d.Id),
		}, nil
	}

	out, err := client.CreateDistributionWithTags(ctx, &cloudfront.CreateDistributionWithTagsInput{
		DistributionConfigWithTags: &types.DistributionConfigWithTags{
			DistributionConfig: distributionConfig,
			Tags: &types.Tags{Items: []types.Tag{
				{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
				{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
			}},
		},
	})
	if err != nil {
		return nil, ui.StopErr(err)
	}

	if err := changeResourceRecordSets(
		ctx,
		cfg,
		subjectAlternativeNames,
		zoneId,
		out.Distribution.DomainName,
	); err != nil {
		return nil, ui.StopErr(err)
	}

	ui.Stop("ok")
	return &Distribution{
		ARN:        aws.ToString(out.Distribution.ARN),
		Comment:    aws.ToString(out.Distribution.DistributionConfig.Comment),
		DomainName: aws.ToString(out.Distribution.DomainName),
		Id:         aws.ToString(out.Distribution.Id),
	}, nil
}

func GetDistributionByName(ctx context.Context, cfg *awscfg.Config, name string) (*Distribution, error) {
	d, err := getDistributionByName(ctx, cfg, name)
	if err != nil {
		return nil, err
	}
	return &Distribution{
		ARN:        aws.ToString(d.ARN),
		Comment:    aws.ToString(d.Comment),
		DomainName: aws.ToString(d.DomainName),
		Id:         aws.ToString(d.Id),
	}, nil
}

func changeResourceRecordSets(
	ctx context.Context,
	cfg *awscfg.Config,
	subjectAlternativeNames []string,
	zoneId string,
	aliasDNSName *string,
) error {
	aliasTarget := &awsroute53.AliasTarget{
		DNSName:              aliasDNSName,
		EvaluateTargetHealth: true,
		HostedZoneId:         aws.String(HostedZoneId),
	}
	var changes []awsroute53.Change
	for _, subjectAlternativeName := range subjectAlternativeNames {
		changes = append(
			changes,
			awsroute53.Change{
				Action: awsroute53.UPSERT,
				ResourceRecordSet: &awsroute53.ResourceRecordSet{
					AliasTarget: aliasTarget,
					Name:        aws.String(subjectAlternativeName),
					Type:        awsroute53.A,
				},
			},
			awsroute53.Change{
				Action: awsroute53.UPSERT,
				ResourceRecordSet: &awsroute53.ResourceRecordSet{
					AliasTarget: aliasTarget,
					Name:        aws.String(subjectAlternativeName),
					Type:        awsroute53.AAAA,
				},
			},
		)
	}
	return awsroute53.ChangeResourceRecordSets(ctx, cfg, zoneId, changes)
}

func getDistributionByName(ctx context.Context, cfg *awscfg.Config, name string) (*types.DistributionSummary, error) {
	distributions, err := listDistributions(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, distribution := range distributions {
		if aws.ToString(distribution.Comment) == name {
			return &distribution, nil
		}
	}
	return nil, fmt.Errorf("distribution %q not found", name)
}

func listDistributions(ctx context.Context, cfg *awscfg.Config) (distributions []types.DistributionSummary, err error) {
	client := cfg.CloudFront()
	var nextMarker *string
	for {
		out, err := client.ListDistributions(ctx, &cloudfront.ListDistributionsInput{
			Marker: nextMarker,
		})
		if err != nil {
			return nil, err
		}
		for _, distribution := range out.DistributionList.Items {
			distributions = append(distributions, distribution)
		}
		if nextMarker = out.DistributionList.NextMarker; nextMarker == nil {
			break
		}
	}
	return
}
