package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/src-bin/substrate/awscfg"
)

type PermissionSet = types.PermissionSet

}

func ListPermissionSets(
	ctx context.Context,
	mgmtCfg *awscfg.Config,
	instance *Instance,
) ([]*PermissionSet, error) {
	client := mgmtCfg.Regional(instance.Region).SSOAdmin()
	var (
		nextToken      *string
		permissionSets []*PermissionSet
	)
	for {
		out, err := client.ListPermissionSets(ctx, &ssoadmin.ListPermissionSetsInput{
			InstanceArn: instance.InstanceArn,
			NextToken:   nextToken,
		})
		if err != nil {
			return nil, err
		}

		for _, permissionSetARN := range out.PermissionSets {
			out, err := client.DescribePermissionSet(ctx, &ssoadmin.DescribePermissionSetInput{
				InstanceArn:      instance.InstanceArn,
				PermissionSetArn: aws.String(permissionSetARN),
			})
			if err != nil {
				return nil, err
			}
			permissionSets = append(permissionSets, out.PermissionSet)
		}

		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return permissionSets, nil
}
