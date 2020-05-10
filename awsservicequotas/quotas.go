package awsservicequotas

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/ui"
)

func EnsureServiceQuota(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
	desiredValue float64,
	// TODO deadline?
) error {

	quota, err := GetServiceQuota(
		svc,
		quotaCode,
		serviceCode,
	)
	if err != nil {
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
	for req := range ListRequestedServiceQuotaChangeHistoryByQuota(
		svc,
		quotaCode,
		serviceCode,
	) {
		if aws.Float64Value(req.DesiredValue) < desiredValue {
			continue
		}
		if status := aws.StringValue(req.Status); status == "PENDING" || status == "CASE_OPENED" {
			ui.Printf(
				"found a previous request to increase service quota %s in %s to %.0f",
				quotaCode,
				aws.StringValue(svc.Client.Config.Region),
				aws.Float64Value(req.DesiredValue),
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
			"requested an increase to service quota %s in %s to %.0f",
			quotaCode,
			aws.StringValue(svc.Client.Config.Region),
			desiredValue,
		)
		log.Printf("%+v", req)
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
	// TODO deadline?
) error {
	ch := make(chan error, len(awsutil.Regions()))

	for _, region := range awsutil.Regions() {
		if awsutil.IsBlacklistedRegion(region) {
			continue
		}
		go func(
			svc *servicequotas.ServiceQuotas,
			quotaCode, serviceCode string,
			desiredValue float64,
			// TODO deadline?
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

	for _, region := range awsutil.Regions() {
		if awsutil.IsBlacklistedRegion(region) {
			continue
		}
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
) chan *servicequotas.RequestedServiceQuotaChange {
	ch := make(chan *servicequotas.RequestedServiceQuotaChange)
	go func(chan<- *servicequotas.RequestedServiceQuotaChange) {
		var nextToken *string
		for {
			in := &servicequotas.ListRequestedServiceQuotaChangeHistoryByQuotaInput{
				NextToken:   nextToken,
				QuotaCode:   aws.String(quotaCode),
				ServiceCode: aws.String(serviceCode),
			}
			out, err := svc.ListRequestedServiceQuotaChangeHistoryByQuota(in)
			if err != nil {
				log.Fatal(err)
			}
			//log.Printf("%+v", out)
			for _, req := range out.RequestedQuotas {
				ch <- req
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
		close(ch)
	}(ch)
	return ch
}

func ListServiceQuotas(
	svc *servicequotas.ServiceQuotas,
	serviceCode string,
) chan *servicequotas.ServiceQuota {
	ch := make(chan *servicequotas.ServiceQuota)
	go func(chan<- *servicequotas.ServiceQuota) {
		var nextToken *string
		for {
			in := &servicequotas.ListServiceQuotasInput{
				NextToken:   nextToken,
				ServiceCode: aws.String(serviceCode),
			}
			out, err := svc.ListServiceQuotas(in)
			if err != nil {
				log.Fatal(err)
			}
			//log.Printf("%+v", out)
			for _, req := range out.Quotas {
				ch <- req
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
		close(ch)
	}(ch)
	return ch
}

func ListServices(
	svc *servicequotas.ServiceQuotas,
) chan *servicequotas.ServiceInfo {
	ch := make(chan *servicequotas.ServiceInfo)
	go func(chan<- *servicequotas.ServiceInfo) {
		var nextToken *string
		for {
			in := &servicequotas.ListServicesInput{
				NextToken: nextToken,
			}
			out, err := svc.ListServices(in)
			if err != nil {
				log.Fatal(err)
			}
			//log.Printf("%+v", out)
			for _, req := range out.Services {
				ch <- req
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
		close(ch)
	}(ch)
	return ch
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
