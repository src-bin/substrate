package awsram

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/src-bin/substrate/awscfg"
)

func EnableSharingWithAwsOrganization(ctx context.Context, cfg *awscfg.Config) error {
	out, err := cfg.RAM().EnableSharingWithAwsOrganization(ctx, &ram.EnableSharingWithAwsOrganizationInput{})
	if err == nil && out != nil && !aws.ToBool(out.ReturnValue) {
		err = errors.New("EnableSharingWithAwsOrganization received ReturnValue: false")
	}
	return err
}
