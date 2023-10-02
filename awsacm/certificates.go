package awsacm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsroute53"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

type CertificateDetail = types.CertificateDetail

func DescribeCertificate(
	ctx context.Context,
	cfg *awscfg.Config,
	certARN string,
) (*CertificateDetail, error) {
	out, err := cfg.ACM().DescribeCertificate(ctx, &acm.DescribeCertificateInput{
		CertificateArn: aws.String(certARN),
	})
	if err != nil {
		return nil, err
	}
	return out.Certificate, nil
}

func EnsureCertificate(
	ctx context.Context,
	cfg *awscfg.Config,
	dnsDomainName string,
	subjectAlternativeNames []string,
	zoneId string,
) (certDetail *CertificateDetail, err error) {
	ui.Spinf("finding or requesting a certificate for %s in %s", dnsDomainName, cfg.Region())
	client := cfg.ACM()

	// Find or create the certificate.
	certSummary, err := getCertificateByName(ctx, cfg, dnsDomainName)
	var certARN string
	if _, ok := err.(NotFound); ok {
		ui.Stop("not found")
		ui.Spinf("requesting a certificate for %s in %s", dnsDomainName, cfg.Region())
		out, err := client.RequestCertificate(ctx, &acm.RequestCertificateInput{
			DomainName:              aws.String(dnsDomainName),
			SubjectAlternativeNames: subjectAlternativeNames,
			Tags: []types.Tag{
				{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
				{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
			},
			ValidationMethod: types.ValidationMethodDns,
		})
		if err != nil {
			return nil, ui.StopErr(err)
		}
		certARN = aws.ToString(out.CertificateArn)
	} else {
		// TODO is there something to meaningfully update?
		certARN = aws.ToString(certSummary.CertificateArn)
	}

	// Await DNS validation record data but return early if it turns out
	// the certificate is already issued, which happens on subsequent runs
	// and in additional regions.
	for range awsutil.StandardJitteredExponentialBackoff() {
		if certDetail, err = DescribeCertificate(ctx, cfg, certARN); err != nil {
			return nil, ui.StopErr(err)
		}
		if certDetail.DomainValidationOptions != nil && certDetail.DomainValidationOptions[0].ResourceRecord != nil {
			break
		}
		if certDetail.Status == types.CertificateStatusIssued {
			return certDetail, ui.StopErr(nil)
		}
	}

	// Validate the request in the DNS.
	ui.Stop("ok")
	ui.Spinf("validating %s", certARN)
	changes := make([]awsroute53.Change, len(certDetail.DomainValidationOptions))
	for i, dvo := range certDetail.DomainValidationOptions {
		changes[i].Action = awsroute53.UPSERT
		changes[i].ResourceRecordSet = &awsroute53.ResourceRecordSet{
			Name:            dvo.ResourceRecord.Name,
			ResourceRecords: []awsroute53.ResourceRecord{{Value: dvo.ResourceRecord.Value}},
			TTL:             aws.Int64(3600),
			Type:            awsroute53.RRType(dvo.ResourceRecord.Type),
		}
	}
	if err := awsroute53.ChangeResourceRecordSets(ctx, cfg, zoneId, changes); err != nil {
		return nil, ui.StopErr(err)
	}

	// Await certificate issuance.
	ui.Stop("ok")
	ui.Spinf("waiting for %s to be issued", certARN)
	for range awsutil.StandardJitteredExponentialBackoff() {
		if certDetail, err = DescribeCertificate(ctx, cfg, certARN); err != nil {
			return nil, ui.StopErr(err)
		}
		if certDetail.Status == types.CertificateStatusIssued {
			return certDetail, ui.StopErr(nil)
		}
	}

	panic("unreachable")
}

type NotFound string

func (err NotFound) Error() string {
	return fmt.Sprintf("certificate for %s not found", string(err))
}

func getCertificateByName(ctx context.Context, cfg *awscfg.Config, dnsDomainName string) (*types.CertificateSummary, error) {
	certSummaries, err := listCertificates(ctx, cfg)
	if err != nil {
		return nil, err
	}
	for _, certSummary := range certSummaries {
		if aws.ToString(certSummary.DomainName) == dnsDomainName {
			return &certSummary, nil
		}
	}
	return nil, NotFound(dnsDomainName)
}

func listCertificates(ctx context.Context, cfg *awscfg.Config) (certSummaries []types.CertificateSummary, err error) {
	client := cfg.ACM()
	var nextToken *string
	for {
		out, err := client.ListCertificates(ctx, &acm.ListCertificatesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, certSummary := range out.CertificateSummaryList {
			certSummaries = append(certSummaries, certSummary)
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
