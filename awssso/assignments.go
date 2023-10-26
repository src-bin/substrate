package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/src-bin/substrate/awscfg"
)

type Assignment struct {
	types.AccountAssignment
	Region string
}

func ListAccountAssignments(
	ctx context.Context,
	mgmtCfg *awscfg.Config,
	instance *Instance,
	permissionSet *PermissionSet,
	accountId string,
) (assignments []*Assignment, err error) {
	client := mgmtCfg.Regional(instance.Region).SSOAdmin()
	var nextToken *string
	for {
		var out *ssoadmin.ListAccountAssignmentsOutput
		out, err = client.ListAccountAssignments(ctx, &ssoadmin.ListAccountAssignmentsInput{
			AccountId:        aws.String(accountId),
			InstanceArn:      instance.InstanceArn,
			NextToken:        nextToken,
			PermissionSetArn: aws.String(permissionSet.ARN),
		})
		if err != nil {
			return
		}
		for _, aa := range out.AccountAssignments {
			assignments = append(assignments, &Assignment{
				AccountAssignment: aa,
				Region:            instance.Region,
			})
		}
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
