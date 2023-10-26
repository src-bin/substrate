package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/regions"
)

const AccessDeniedException = "AccessDeniedException"

type Instance struct {
	types.InstanceMetadata
	Region string
}

func ListInstances(ctx context.Context, mgmtCfg *awscfg.Config) (instances []*Instance, err error) {
	for _, region := range regions.Selected() {
		client := mgmtCfg.Regional(region).SSOAdmin()
		var nextToken *string
		for {
			var out *ssoadmin.ListInstancesOutput
			out, err = client.ListInstances(ctx, &ssoadmin.ListInstancesInput{
				NextToken: nextToken,
			})
			if awsutil.ErrorCodeIs(err, AccessDeniedException) { // hostile lie of a response instead of just [] (or not but we can't tell)
				err = nil
				break
			} else if err != nil {
				return
			}
			for _, im := range out.Instances {
				instances = append(instances, &Instance{
					InstanceMetadata: im,
					Region:           region,
				})
			}
			if nextToken = out.NextToken; nextToken == nil {
				break
			}
		}
	}
	return
}
