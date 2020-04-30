package awsiam

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsutil"
)

type SAMLProvider struct {
	Arn string
}

func EnsureSAMLProvider(svc *iam.IAM, name, metadata string) (*SAMLProvider, error) {

	out, err := createSAMLProvider(svc, name, metadata)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {

		providers := listSAMLProviders(svc)
		for _, provider := range providers {
			arn := aws.StringValue(provider.Arn)
			if strings.HasSuffix(arn, "/"+name) {
				out, err = updateSAMLProvider(svc, arn, metadata)
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

func createSAMLProvider(svc *iam.IAM, name, metadata string) (*SAMLProvider, error) {
	in := &iam.CreateSAMLProviderInput{
		Name:                 aws.String(name),
		SAMLMetadataDocument: aws.String(metadata),
	}
	out, err := svc.CreateSAMLProvider(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &SAMLProvider{aws.StringValue(out.SAMLProviderArn)}, nil
}

func listSAMLProviders(svc *iam.IAM) []*iam.SAMLProviderListEntry {
	out, err := svc.ListSAMLProviders(&iam.ListSAMLProvidersInput{})
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", out)
	return out.SAMLProviderList
}

func updateSAMLProvider(svc *iam.IAM, arn, metadata string) (*SAMLProvider, error) {
	in := &iam.UpdateSAMLProviderInput{
		SAMLMetadataDocument: aws.String(metadata),
		SAMLProviderArn:      aws.String(arn),
	}
	out, err := svc.UpdateSAMLProvider(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return &SAMLProvider{aws.StringValue(out.SAMLProviderArn)}, nil
}
