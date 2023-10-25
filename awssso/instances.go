package awssso

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/roles"
)

const AccessDeniedException = "AccessDeniedException"

type InstanceMetadata struct {
	types.InstanceMetadata
	AccountId, Region string
}

func ListInstances(ctx context.Context, substrateCfg *awscfg.Config) ([]InstanceMetadata, error) {
	instances := []InstanceMetadata{}
	listInstances := func(client *ssoadmin.Client, accountId, region string) error {
		var nextToken *string
		for {
			out, err := client.ListInstances(ctx, &ssoadmin.ListInstancesInput{
				NextToken: nextToken,
			})
			if awsutil.ErrorCodeIs(err, AccessDeniedException) { // hostile lie of a response instead of just []
				break
			} else if err != nil {
				return err
			}
			for _, im := range out.Instances {
				instances = append(instances, InstanceMetadata{
					AccountId:        accountId,
					InstanceMetadata: im,
					Region:           region,
				})
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
		return nil
	}

	substrateAccountId, err := substrateCfg.AccountId(ctx)
	if err != nil {
		return nil, err
	}
	if err := listInstances(substrateCfg.SSOAdmin(), substrateAccountId, substrateCfg.Region()); err != nil {
		return nil, err
	}

	mgmtCfg, err := substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour)
	if err != nil {
		return nil, err
	}
	mgmtAccountId, err := mgmtCfg.AccountId(ctx)
	if err != nil {
		return nil, err
	}
	if err := listInstances(mgmtCfg.SSOAdmin(), mgmtAccountId, mgmtCfg.Region()); err != nil {
		return nil, err
	}

	return instances, nil
}
