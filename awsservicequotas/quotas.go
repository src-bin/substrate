package awsservicequotas

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicequotas"
)

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
	return out.Quota, nil
}

func RequestServiceQuotaIncrease(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
	desiredValue float64,
) (*servicequotas.RequestedServiceQuotaChange, error) {
	in := &servicequotas.RequestServiceQuotaIncreaseInput{
		QuotaCode:   aws.String(quotaCode),
		ServiceCode: aws.String(serviceCode),
	}
	out, err := svc.RequestServiceQuotaIncrease(in)
	if err != nil {
		return nil, err
	}
	return out.RequestedQuota, nil
}
