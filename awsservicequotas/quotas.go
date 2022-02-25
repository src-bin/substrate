package awsservicequotas

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

const (
	APPROVED    = "APPROVED"
	CASE_OPENED = "CASE_OPENED"
	PENDING     = "PENDING"

	NoSuchResourceException = "NoSuchResourceException"
)

type DeadlinePassed struct{ QuotaCode, ServiceCode string }

func (err DeadlinePassed) Error() string {
	return fmt.Sprintf("deadline passed raising quota %s; continuing", err.QuotaCode)
}

// EnsureServiceQuota tries to find the current value of the given quota (or
// its default, if the true current value isn't available) and raise it to
// desiredValue if it is (believed to be) lower than requiredValue. The
// separation of desiredValue from requiredValue enables callers to raise
// limits only when necessary and to raise them far enough that foreseeable
// future calls don't need to raise the limit again.
//
// Limits for which only the default value is available are awkward to use
// once the limit has been raised from the default value. In these cases it's
// best to attempt to create the resource, receive a LimitExceeded (or similar)
// exception, and then call EnsureServiceQuota.
func EnsureServiceQuota(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
	requiredValue, desiredValue float64,
	deadline time.Time,
) error {

	quota, err := GetServiceQuota(
		svc,
		quotaCode,
		serviceCode,
	)
	if awsutil.ErrorCodeIs(err, NoSuchResourceException) {
		quota, err = GetAWSDefaultServiceQuota(
			svc,
			quotaCode,
			serviceCode,
		)
	}
	if awsutil.ErrorCodeIs(err, NoSuchResourceException) {
		return nil // the presumption being we don't need to raise limits we can't see
	}
	if err != nil {
		ui.Print(aws.StringValue(svc.Client.Config.Region), quotaCode, serviceCode)
		return err
	}
	//log.Printf("%+v", quota)

	if aws.Float64Value(quota.Value) >= requiredValue {
		ui.Printf(
			"service quota %s in %s is already %.0f >= %.0f",
			quotaCode,
			aws.StringValue(svc.Client.Config.Region),
			aws.Float64Value(quota.Value),
			requiredValue,
		)
		return nil
	}

	if desiredValue < requiredValue {
		ui.Print(
			"desired quota value %.0f < required quota value %.0f; raising quota to %.0f",
			desiredValue,
			requiredValue,
			requiredValue,
		)
		desiredValue = requiredValue
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
		if status := aws.StringValue(change.Status); status == PENDING || status == CASE_OPENED {
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

	var zero time.Time
	for {

		// Check the deadline first so that calls can set a deadline in the
		// past to get quick is-the-limit-sufficient feedback.
		if deadline != zero && time.Now().After(deadline) {
			return DeadlinePassed{quotaCode, serviceCode}
		}

		// Sleep before the real work so we can `continue` without risk of
		// being throttled.
		time.Sleep(time.Minute)

		// First try to directly query the limit, which may not work because
		// some limits aren't visible to Service Quotas.
		quota, err := GetServiceQuota(
			svc,
			quotaCode,
			serviceCode,
		)
		if awsutil.ErrorCodeIs(err, NoSuchResourceException) {
			// This is an invisible limit. GetServiceQuota can't break out of
			// the loop so we fall through.
		} else if awsutil.ErrorCodeIs(err, awsutil.RequestError) {
			continue
		} else if err != nil {
			return err
		} else if value := aws.Float64Value(quota.Value); value >= desiredValue {
			break
		}

		// If that didn't work, try to see if the limit increase was approved,
		// which would be a good sign but doesn't definitely mean the
		// operation's going to work immediately.
		changes, err := ListRequestedServiceQuotaChangeHistoryByQuota(
			svc,
			quotaCode,
			serviceCode,
		)
		if awsutil.ErrorCodeIs(err, awsutil.RequestError) {
			continue
		} else if err != nil {
			return err
		}
		for _, change := range changes {
			if aws.Float64Value(change.DesiredValue) < desiredValue {
				continue
			}
			if status := aws.StringValue(change.Status); status == APPROVED {
				break
			}
		}

	}

	ui.Printf(
		"service quota %s in %s increased to %.0f",
		quotaCode,
		aws.StringValue(svc.Client.Config.Region),
		desiredValue,
	)
	return nil
}

func EnsureServiceQuotaInAllRegions(
	sess *session.Session,
	quotaCode, serviceCode string,
	requiredValue, desiredValue float64,
	deadline time.Time,
) error {
	ch := make(chan error, len(regions.Selected()))

	for _, region := range regions.Selected() {
		go func(
			svc *servicequotas.ServiceQuotas,
			quotaCode, serviceCode string,
			desiredValue float64,
			deadline time.Time,
			ch chan<- error,
		) {
			ch <- EnsureServiceQuota(svc, quotaCode, serviceCode, requiredValue, desiredValue, deadline)
		}(
			servicequotas.New(
				sess,
				&aws.Config{Region: aws.String(region)},
			),
			quotaCode,
			serviceCode,
			desiredValue,
			deadline,
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

func GetAWSDefaultServiceQuota(
	svc *servicequotas.ServiceQuotas,
	quotaCode, serviceCode string,
) (*servicequotas.ServiceQuota, error) {
	in := &servicequotas.GetAWSDefaultServiceQuotaInput{
		QuotaCode:   aws.String(quotaCode),
		ServiceCode: aws.String(serviceCode),
	}
	out, err := svc.GetAWSDefaultServiceQuota(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.Quota, nil
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

// NewGlobal creates a client from a session, unconditionally in us-east-1
// because Service Quota appears to just require quotas for global services
// to be inspected and manipulated from us-east-1. Hateful.
func NewGlobal(sess *session.Session) *servicequotas.ServiceQuotas {
	return servicequotas.New(sess, &aws.Config{Region: aws.String("us-east-1")})
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
