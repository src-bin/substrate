package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const SessionDuration = "PT12H"

type PermissionSet = types.PermissionSet

func EnsurePermissionSet(
	ctx context.Context,
	mgmtCfg *awscfg.Config,
	instance *Instance,
	name string,
	awsManagedPolicyARNs, customerManagedPolicyNames []string,
	inlinePolicyDoc *policies.Document,
) (*PermissionSet, error) {
	client := mgmtCfg.Regional(instance.Region).SSOAdmin()
	tags := []types.Tag{
		{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
		{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
	}
	var permissionSet *PermissionSet
	if out, err := client.CreatePermissionSet(ctx, &ssoadmin.CreatePermissionSetInput{
		InstanceArn:     instance.InstanceArn,
		Name:            aws.String(name),
		SessionDuration: aws.String(SessionDuration),
		Tags:            tags,
	}); err == nil {
		permissionSet = out.PermissionSet
	} else if awsutil.ErrorCodeIs(err, ConflictException) {
		permissionSets, err := ListPermissionSets(ctx, mgmtCfg, instance)
		if err != nil {
			return nil, err
		}
		for _, ps := range permissionSets {
			if aws.ToString(ps.Name) == name {
				if _, err := client.UpdatePermissionSet(ctx, &ssoadmin.UpdatePermissionSetInput{
					InstanceArn:      instance.InstanceArn,
					PermissionSetArn: ps.PermissionSetArn,
					SessionDuration:  aws.String(SessionDuration),
				}); err != nil {
					return nil, err
				}
				permissionSet = ps
			}
		}
		if permissionSet == nil {
			return nil, NotFound{"permission set", name}
		}
		if _, err := client.TagResource(ctx, &ssoadmin.TagResourceInput{
			InstanceArn: instance.InstanceArn,
			ResourceArn: permissionSet.PermissionSetArn,
			Tags:        tags,
		}); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	ui.Debug(permissionSet)

	for _, awsManagedPolicyARN := range awsManagedPolicyARNs {
		if _, err := client.AttachManagedPolicyToPermissionSet(ctx, &ssoadmin.AttachManagedPolicyToPermissionSetInput{
			InstanceArn:      instance.InstanceArn,
			ManagedPolicyArn: aws.String(awsManagedPolicyARN),
			PermissionSetArn: permissionSet.PermissionSetArn,
		}); err != nil && !awsutil.ErrorCodeIs(err, ConflictException) {
			return nil, err
		}
	}

	for _, customerManagedPolicyName := range customerManagedPolicyNames {
		if _, err := client.AttachCustomerManagedPolicyReferenceToPermissionSet(
			ctx,
			&ssoadmin.AttachCustomerManagedPolicyReferenceToPermissionSetInput{
				CustomerManagedPolicyReference: &types.CustomerManagedPolicyReference{
					Name: aws.String(customerManagedPolicyName),
				},
				InstanceArn:      instance.InstanceArn,
				PermissionSetArn: permissionSet.PermissionSetArn,
			},
		); err != nil && !awsutil.ErrorCodeIs(err, ConflictException) {
			return nil, err
		}
	}

	if inlinePolicyDoc != nil {
		inlinePolicy, err := inlinePolicyDoc.Marshal()
		if err != nil {
			return nil, err
		}
		if _, err := client.PutInlinePolicyToPermissionSet(ctx, &ssoadmin.PutInlinePolicyToPermissionSetInput{
			InlinePolicy:     aws.String(inlinePolicy),
			InstanceArn:      instance.InstanceArn,
			PermissionSetArn: permissionSet.PermissionSetArn,
		}); err != nil {
			return nil, err
		}
	}

	return permissionSet, nil
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
