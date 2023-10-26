package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/src-bin/substrate/awscfg"
)

type PermissionSet struct {
	ARN, Region string
}

func ListPermissionSets(
	ctx context.Context,
	mgmtCfg *awscfg.Config,
	instance *Instance,
) (permissionSets []*PermissionSet, err error) {
	client := mgmtCfg.Regional(instance.Region).SSOAdmin()
	var nextToken *string
	for {
		var out *ssoadmin.ListPermissionSetsOutput
		if out, err = client.ListPermissionSets(ctx, &ssoadmin.ListPermissionSetsInput{
			InstanceArn: instance.InstanceArn,
			NextToken:   nextToken,
		}); err != nil {
			return
		}
		for _, permissionSetARN := range out.PermissionSets {
			permissionSets = append(permissionSets, &PermissionSet{
				ARN:    permissionSetARN,
				Region: instance.Region,
			})
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
