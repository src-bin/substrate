package awsservicequotas

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

const NoSuchResourceException = "NoSuchResourceException"

func EnsureServiceQuota(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
	desiredValue float64,
) error {

	quota, err := GetServiceQuota(
		svc,
		quotaCode,
		serviceCode,
	)
	if awsutil.ErrorCodeIs(err, NoSuchResourceException) {
		return nil // the presumption being we don't need to raise limits we can't see
	}
	if err != nil {
		log.Println(aws.StringValue(svc.Client.Config.Region), quotaCode, serviceCode)
		return err
	}
	//log.Printf("%+v", quota)

	if aws.Float64Value(quota.Value) >= desiredValue {
		ui.Printf(
			"service quota %s in %s is already %.0f",
			quotaCode,
			aws.StringValue(svc.Client.Config.Region),
			aws.Float64Value(quota.Value),
		)
		return nil
	}

	requested := false
	changes, err := ListRequestedServiceQuotaChangeHistoryByQuota(
		svc,
		quotaCode,
		serviceCode,
	)
	if err != nil {
		return err
	}
	for _, change := range changes {
		if aws.Float64Value(change.DesiredValue) < desiredValue {
			continue
		}
		if status := aws.StringValue(change.Status); status == "PENDING" || status == "CASE_OPENED" {
			ui.Printf(
				"found a previous request to increase service quota %s in %s to %.0f; waiting for it to be resolved",
				quotaCode,
				aws.StringValue(svc.Client.Config.Region),
				aws.Float64Value(change.DesiredValue),
			)
			requested = true
		}
	}

	if !requested {
		req, err := RequestServiceQuotaIncrease(
			svc,
			quotaCode,
			serviceCode,
			desiredValue,
		)
		if err != nil {
			return err
		}
		ui.Printf(
			"requested an increase to service quota %s in %s to %.0f; waiting for it to be resolved",
			req.QuotaCode,
			svc.Client.Config.Region,
			aws.Float64Value(req.DesiredValue),
		)
		//log.Printf("%+v", req)
	}

	for {
		quota, err := GetServiceQuota(
			svc,
			quotaCode,
			serviceCode,
		)
		if err != nil {
			log.Print(aws.StringValue(svc.Client.Config.Region))
			return err
		}
		//log.Printf("%+v", quota)
		if value := aws.Float64Value(quota.Value); value >= desiredValue {
			ui.Printf(
				"received an increase to service quota %s in %s to %.0f",
				quotaCode,
				aws.StringValue(svc.Client.Config.Region),
				value,
			)
			break
		}
		time.Sleep(time.Minute)
	}

	return nil
}

func EnsureServiceQuotaInAllRegions(
	sess *session.Session,
	quotaCode, serviceCode string,
	desiredValue float64,
) error {
	ch := make(chan error, len(regions.Selected()))

	for _, region := range regions.Selected() {
		go func(
			svc *servicequotas.ServiceQuotas,
			quotaCode, serviceCode string,
			desiredValue float64,
			ch chan<- error,
		) {
			ch <- EnsureServiceQuota(svc, quotaCode, serviceCode, desiredValue)
		}(
			servicequotas.New(
				sess,
				&aws.Config{Region: aws.String(region)},
			),
			quotaCode,
			serviceCode,
			desiredValue,
			ch,
		)
	}

	for range regions.Selected() {
		if err := <-ch; err != nil {
			return err
		}
	}

	ui.Printf(
		"service quota %s is at least %.0f in all regions",
		quotaCode,
		desiredValue,
	)
	return nil
}

func GetServiceQuota(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
) (*servicequotas.ServiceQuota, error) {
	in := &servicequotas.GetServiceQuotaInput{
		QuotaCode:   aws.String(quotaCode),
		ServiceCode: aws.String(serviceCode),
	}
	out, err := svc.GetServiceQuota(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.Quota, nil
}

func ListRequestedServiceQuotaChangeHistoryByQuota(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
) (changes []*servicequotas.RequestedServiceQuotaChange, err error) {
	var nextToken *string
	for {
		in := &servicequotas.ListRequestedServiceQuotaChangeHistoryByQuotaInput{
			NextToken:   nextToken,
			QuotaCode:   aws.String(quotaCode),
			ServiceCode: aws.String(serviceCode),
		}
		out, err := svc.ListRequestedServiceQuotaChangeHistoryByQuota(in)
		if err != nil {
			return nil, err
		}
		//log.Printf("%+v", out)
		changes = append(changes, out.RequestedQuotas...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func ListServiceQuotas(
	svc *servicequotas.ServiceQuotas,
	serviceCode string,
) (quotas []*servicequotas.ServiceQuota, err error) {
	var nextToken *string
	for {
		in := &servicequotas.ListServiceQuotasInput{
			NextToken:   nextToken,
			ServiceCode: aws.String(serviceCode),
		}
		out, err := svc.ListServiceQuotas(in)
		if err != nil {
			return nil, err
		}
		//log.Printf("%+v", out)
		quotas = append(quotas, out.Quotas...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func ListServices(
	svc *servicequotas.ServiceQuotas,
) (services []*servicequotas.ServiceInfo, err error) {
	var nextToken *string
	for {
		in := &servicequotas.ListServicesInput{
			NextToken: nextToken,
		}
		out, err := svc.ListServices(in)
		if err != nil {
			return nil, err
		}
		//log.Printf("%+v", out)
		services = append(services, out.Services...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}

func RequestServiceQuotaIncrease(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
	desiredValue float64,
) (*servicequotas.RequestedServiceQuotaChange, error) {
	in := &servicequotas.RequestServiceQuotaIncreaseInput{
		DesiredValue: aws.Float64(desiredValue),
		QuotaCode:    aws.String(quotaCode),
		ServiceCode:  aws.String(serviceCode),
	}
	out, err := svc.RequestServiceQuotaIncrease(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.RequestedQuota, nil
}
