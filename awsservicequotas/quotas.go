package awsservicequotas

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

const NoSuchResourceException = "NoSuchResourceException"

type (
	RequestedServiceQuotaChange = types.RequestedServiceQuotaChange
	ServiceInfo                 = types.ServiceInfo
	ServiceQuota                = types.ServiceQuota
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
	ctx context.Context,
	cfg *awscfg.Config,
	quotaCode, serviceCode string,
	requiredValue, desiredValue float64,
	deadline time.Time,
) error {

	quota, err := GetServiceQuota(
		ctx,
		cfg,
		quotaCode,
		serviceCode,
	)
	if awsutil.ErrorCodeIs(err, NoSuchResourceException) {
		quota, err = GetAWSDefaultServiceQuota(
			ctx,
			cfg,
			quotaCode,
			serviceCode,
		)
	}
	if awsutil.ErrorCodeIs(err, NoSuchResourceException) {
		return nil // the presumption being we don't need to raise limits we can't see
	}
	if err != nil {
		ui.Print(cfg.Region(), quotaCode, serviceCode)
		return err
	}
	//log.Printf("%+v", quota)

	if aws.ToFloat64(quota.Value) >= requiredValue {
		ui.Printf(
			"service quota %s in %s is already %.0f >= %.0f",
			quotaCode,
			cfg.Region(),
			aws.ToFloat64(quota.Value),
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
		ctx,
		cfg,
		quotaCode,
		serviceCode,
	)
	if err != nil {
		return err
	}
	for _, change := range changes {
		if aws.ToFloat64(change.DesiredValue) < desiredValue {
			continue
		}
		if status := change.Status; status == types.RequestStatusPending || status == types.RequestStatusCaseOpened {
			ui.Printf(
				"found a previous request to increase service quota %s in %s to %.0f; waiting for it to be resolved",
				quotaCode,
				cfg.Region(),
				aws.ToFloat64(change.DesiredValue),
			)
			requested = true
		}
	}

	if !requested {
		req, err := RequestServiceQuotaIncrease(
			ctx,
			cfg,
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
			cfg.Region(),
			aws.ToFloat64(req.DesiredValue),
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
			ctx,
			cfg,
			quotaCode,
			serviceCode,
		)
		//log.Printf("%+v %v", quota, err)
		if awsutil.ErrorCodeIs(err, NoSuchResourceException) {
			// This is an invisible limit. GetServiceQuota can't break out of
			// the loop so we fall through.
		} else if awsutil.ErrorCodeIs(err, awsutil.RequestError) {
			//log.Print(err)
			continue
		} else if err != nil {
			return err
		} else if value := aws.ToFloat64(quota.Value); value >= desiredValue {
			break
		}

		// If that didn't work, try to see if the limit increase was approved,
		// which would be a good sign but doesn't definitely mean the
		// operation's going to work immediately.
		changes, err := ListRequestedServiceQuotaChangeHistoryByQuota(
			ctx,
			cfg,
			quotaCode,
			serviceCode,
		)
		//log.Printf("%+v %v", changes, err)
		if awsutil.ErrorCodeIs(err, awsutil.RequestError) {
			//log.Print(err)
			continue
		} else if err != nil {
			return err
		}
		for _, change := range changes {
			if aws.ToFloat64(change.DesiredValue) < desiredValue {
				continue
			}
			if status := change.Status; status == types.RequestStatusApproved || status == types.RequestStatusCaseClosed {
				break
			}
		}

	}

	ui.Printf(
		"service quota %s in %s increased to %.0f",
		quotaCode,
		cfg.Region(),
		desiredValue,
	)
	return nil
}

func EnsureServiceQuotaInAllRegions(
	ctx context.Context,
	cfg *awscfg.Config,
	quotaCode, serviceCode string,
	requiredValue, desiredValue float64,
	deadline time.Time,
) error {
	ch := make(chan error, len(regions.Selected()))

	for _, region := range regions.Selected() {
		go func(
			ctx context.Context,
			cfg *awscfg.Config,
			quotaCode, serviceCode string,
			desiredValue float64,
			deadline time.Time,
			ch chan<- error,
		) {
			ch <- EnsureServiceQuota(ctx, cfg, quotaCode, serviceCode, requiredValue, desiredValue, deadline)
		}(
			ctx,
			cfg.Regional(region),
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
	ctx context.Context,
	cfg *awscfg.Config,
	quotaCode, serviceCode string,
) (*ServiceQuota, error) {
	out, err := cfg.ServiceQuotas().GetAWSDefaultServiceQuota(ctx, &servicequotas.GetAWSDefaultServiceQuotaInput{
		QuotaCode:   aws.String(quotaCode),
		ServiceCode: aws.String(serviceCode),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.Quota, nil
}

func GetServiceQuota(
	ctx context.Context,
	cfg *awscfg.Config,
	quotaCode, serviceCode string,
) (*ServiceQuota, error) {
	out, err := cfg.ServiceQuotas().GetServiceQuota(ctx, &servicequotas.GetServiceQuotaInput{
		QuotaCode:   aws.String(quotaCode),
		ServiceCode: aws.String(serviceCode),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.Quota, nil
}

func ListRequestedServiceQuotaChangeHistoryByQuota(
	ctx context.Context,
	cfg *awscfg.Config,
	quotaCode, serviceCode string,
) (changes []RequestedServiceQuotaChange, err error) {
	var nextToken *string
	for {
		out, err := cfg.ServiceQuotas().ListRequestedServiceQuotaChangeHistoryByQuota(
			ctx,
			&servicequotas.ListRequestedServiceQuotaChangeHistoryByQuotaInput{
				NextToken:   nextToken,
				QuotaCode:   aws.String(quotaCode),
				ServiceCode: aws.String(serviceCode),
			},
		)
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
	ctx context.Context,
	cfg *awscfg.Config,
	serviceCode string,
) (quotas []ServiceQuota, err error) {
	var nextToken *string
	for {
		out, err := cfg.ServiceQuotas().ListServiceQuotas(ctx, &servicequotas.ListServiceQuotasInput{
			NextToken:   nextToken,
			ServiceCode: aws.String(serviceCode),
		})
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
	ctx context.Context,
	cfg *awscfg.Config,
) (services []ServiceInfo, err error) {
	var nextToken *string
	for {
		out, err := cfg.ServiceQuotas().ListServices(ctx, &servicequotas.ListServicesInput{
			NextToken: nextToken,
		})
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
	ctx context.Context,
	cfg *awscfg.Config,
	quotaCode, serviceCode string,
	desiredValue float64,
) (*RequestedServiceQuotaChange, error) {
	out, err := cfg.ServiceQuotas().RequestServiceQuotaIncrease(ctx, &servicequotas.RequestServiceQuotaIncreaseInput{
		DesiredValue: aws.Float64(desiredValue),
		QuotaCode:    aws.String(quotaCode),
		ServiceCode:  aws.String(serviceCode),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.RequestedQuota, nil
}
