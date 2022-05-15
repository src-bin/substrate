package awsiam

import (
	"context"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	iamv1 "github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

type SAMLProvider struct {
	Arn string
}

func EnsureSAMLProvider(
	ctx context.Context,
	cfg *awscfg.Config,
	name, metadata string,
) (*SAMLProvider, error) {

	out, err := createSAMLProvider(ctx, cfg, name, metadata)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {

		providers := listSAMLProviders(ctx, cfg)
		for _, provider := range providers {
			arn := aws.ToString(provider.Arn)
			if strings.HasSuffix(arn, "/"+name) {
				out, err = updateSAMLProvider(ctx, cfg, arn, metadata)
			}
		}

	}
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)

	// TODO tag the SAML provider

	return out, nil
}

func EnsureSAMLProviderV1(
	svc *iamv1.IAM,
	name, metadata string,
) (*SAMLProvider, error) {

	out, err := createSAMLProviderV1(svc, name, metadata)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {

		providers := listSAMLProvidersV1(svc)
		for _, provider := range providers {
			arn := aws.ToString(provider.Arn)
			if strings.HasSuffix(arn, "/"+name) {
				out, err = updateSAMLProviderV1(svc, arn, metadata)
			}
		}

	}
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)

	// TODO tag the SAML provider

	return out, nil
}

func createSAMLProvider(
	ctx context.Context,
	cfg *awscfg.Config,
	name, metadata string,
) (*SAMLProvider, error) {
	out, err := cfg.ClientForIAM().CreateSAMLProvider(ctx, &iam.CreateSAMLProviderInput{
		Name:                 aws.String(name),
		SAMLMetadataDocument: aws.String(metadata),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &SAMLProvider{aws.ToString(out.SAMLProviderArn)}, nil
}

func createSAMLProviderV1(
	svc *iamv1.IAM,
	name, metadata string,
) (*SAMLProvider, error) {
	out, err := svc.CreateSAMLProvider(&iamv1.CreateSAMLProviderInput{
		Name:                 aws.String(name),
		SAMLMetadataDocument: aws.String(metadata),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &SAMLProvider{aws.ToString(out.SAMLProviderArn)}, nil
}

func listSAMLProviders(ctx context.Context, cfg *awscfg.Config) []types.SAMLProviderListEntry {
	out, err := cfg.ClientForIAM().ListSAMLProviders(ctx, &iam.ListSAMLProvidersInput{})
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", out)
	return out.SAMLProviderList
}

func listSAMLProvidersV1(svc *iamv1.IAM) []*iamv1.SAMLProviderListEntry {
	out, err := svc.ListSAMLProviders(&iamv1.ListSAMLProvidersInput{})
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", out)
	return out.SAMLProviderList
}

func updateSAMLProvider(
	ctx context.Context,
	cfg *awscfg.Config,
	arn, metadata string,
) (*SAMLProvider, error) {
	out, err := cfg.ClientForIAM().UpdateSAMLProvider(ctx, &iam.UpdateSAMLProviderInput{
		SAMLMetadataDocument: aws.String(metadata),
		SAMLProviderArn:      aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &SAMLProvider{aws.ToString(out.SAMLProviderArn)}, nil
}

func updateSAMLProviderV1(
	svc *iamv1.IAM,
	arn, metadata string,
) (*SAMLProvider, error) {
	out, err := svc.UpdateSAMLProvider(&iamv1.UpdateSAMLProviderInput{
		SAMLMetadataDocument: aws.String(metadata),
		SAMLProviderArn:      aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &SAMLProvider{aws.ToString(out.SAMLProviderArn)}, nil
}
