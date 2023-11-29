package awsram

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/ram/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/tagging"
)

type ResourceShare = types.ResourceShare

func EnableSharingWithAwsOrganization(ctx context.Context, cfg *awscfg.Config) error {
	out, err := cfg.RAM().EnableSharingWithAwsOrganization(ctx, &ram.EnableSharingWithAwsOrganizationInput{})
	if err == nil && out != nil && !aws.ToBool(out.ReturnValue) {
		err = errors.New("EnableSharingWithAwsOrganization received ReturnValue: false")
	}
	return err
}

func EnsureResourceShare(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	principals, resourceARNs []string,
	tags tagging.Map,
) (*ResourceShare, error) {
	client := cfg.RAM()
	if rs, err := GetResourceShare(ctx, cfg, name); err == nil { // check first because...
		return rs, nil
	} else if err != nil {
		if _, ok := err.(NotFound); !ok {
			return nil, err
		}
	}
	if out, err := client.CreateResourceShare(ctx, &ram.CreateResourceShareInput{
		AllowExternalPrincipals: aws.Bool(false),
		Name:                    aws.String(name), // ...this is not a unique key
		Principals:              principals,
		ResourceArns:            resourceARNs,
		Tags:                    tagStructs(tags),
	}); err == nil {
		return out.ResourceShare, nil
	} else if err != nil {
		return nil, err
	}
	rs, err := GetResourceShare(ctx, cfg, name)
	if err != nil {
		return nil, err
	}
	if _, err := client.AssociateResourceShare(ctx, &ram.AssociateResourceShareInput{
		Principals:       principals,
		ResourceArns:     resourceARNs,
		ResourceShareArn: rs.ResourceShareArn,
	}); err != nil {
		return nil, err
	}
	return GetResourceShare(ctx, cfg, name) // refresh
}

func GetResourceShare(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
) (*ResourceShare, error) {
	out, err := cfg.RAM().GetResourceShares(ctx, &ram.GetResourceSharesInput{
		Name:          aws.String(name),
		ResourceOwner: types.ResourceOwnerSelf,
	})
	if err != nil {
		return nil, err
	}
	if len(out.ResourceShares) < 1 {
		return nil, NotFound(name)
	}
	rs := out.ResourceShares[0] // don't leak the whole slice
	return &rs, nil
}

type NotFound string

func (err NotFound) Error() string {
	return fmt.Sprintf("resource share %q not found", string(err))
}
