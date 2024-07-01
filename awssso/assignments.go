package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/src-bin/substrate/awscfg"
)

type AccountAssignment = types.AccountAssignment

func EnsureGroupAccountAssignment(
	ctx context.Context,
	mgmtCfg *awscfg.Config,
	instance *Instance,
	permissionSet *PermissionSet,
	accountId, groupId string,
) error {
	client := mgmtCfg.Regional(instance.Region).SSOAdmin()
	_, err := client.CreateAccountAssignment(ctx, &ssoadmin.CreateAccountAssignmentInput{
		InstanceArn:      instance.InstanceArn,
		PermissionSetArn: permissionSet.PermissionSetArn,
		PrincipalId:      aws.String(groupId),
		PrincipalType:    types.PrincipalTypeGroup,
		TargetId:         aws.String(accountId),
		TargetType:       types.TargetTypeAwsAccount,
	})
	return err
}

func ListAccountAssignments(
	ctx context.Context,
	mgmtCfg *awscfg.Config,
	instance *Instance,
	permissionSet *PermissionSet,
	accountId string,
) (assignments []AccountAssignment, err error) {
	client := mgmtCfg.Regional(instance.Region).SSOAdmin()
	var nextToken *string
	for {
		var out *ssoadmin.ListAccountAssignmentsOutput
		out, err = client.ListAccountAssignments(ctx, &ssoadmin.ListAccountAssignmentsInput{
			AccountId:        aws.String(accountId),
			InstanceArn:      instance.InstanceArn,
			NextToken:        nextToken,
			PermissionSetArn: permissionSet.PermissionSetArn,
		})
		if err != nil {
			return
		}
		assignments = append(assignments, out.AccountAssignments...)
		if nextToken = out.NextToken; nextToken == nil {
			break
		}
	}
	return
}
