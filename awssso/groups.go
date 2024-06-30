package awssso

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/identitystore"
	"github.com/aws/aws-sdk-go-v2/service/identitystore/document"
	"github.com/aws/aws-sdk-go-v2/service/identitystore/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
)

type Group struct{ GroupId, IdentityStoreId, Name string }

func EnsureGroup(
	ctx context.Context,
	mgmtCfg *awscfg.Config,
	instance *Instance,
	name string,
) (*Group, error) {
	client := mgmtCfg.Regional(instance.Region).IdentityStore()

	if out, err := client.CreateGroup(ctx, &identitystore.CreateGroupInput{
		IdentityStoreId: instance.IdentityStoreId,
		DisplayName:     aws.String(name),
	}); err == nil {
		return &Group{
			GroupId:         aws.ToString(out.GroupId),
			IdentityStoreId: aws.ToString(instance.IdentityStoreId),
			Name:            name,
		}, nil
	} else if !awsutil.ErrorCodeIs(err, ConflictException) {
		return nil, err
	}

	out, err := client.GetGroupId(ctx, &identitystore.GetGroupIdInput{
		AlternateIdentifier: &types.AlternateIdentifierMemberUniqueAttribute{Value: types.UniqueAttribute{
			AttributePath:  aws.String("displayName"),
			AttributeValue: document.NewLazyDocument(name),
		}},
		IdentityStoreId: instance.IdentityStoreId,
	})
	if err != nil {
		return nil, err
	}
	return &Group{
		GroupId:         aws.ToString(out.GroupId),
		IdentityStoreId: aws.ToString(instance.IdentityStoreId),
		Name:            name,
	}, nil
}
