package awsiam

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	CreateAccessKeyTriesBeforeDeleteAll = 4
	CreateAccessKeyTriesTotal           = 8
)

func AllDayCredentials(
	ctx context.Context,
	cfg *awscfg.Config,
	roleName string,
) (creds aws.Credentials, err error) {
	var accessKey *types.AccessKey
	for i := 0; i < CreateAccessKeyTriesTotal; i++ {
		accessKey, err = CreateAccessKey(ctx, cfg, users.CredentialFactory)
		if awsutil.ErrorCodeIs(err, LimitExceeded) {
			if i == CreateAccessKeyTriesBeforeDeleteAll {
				if err = DeleteAllAccessKeys(ctx, cfg, users.CredentialFactory); err != nil {
					return
				}
			}
			continue
		}
		break
	}
	if err != nil {
		return
	}
	defer func() {
		if err := DeleteAccessKey(
			ctx,
			cfg,
			users.CredentialFactory,
			aws.ToString(accessKey.AccessKeyId),
		); err != nil {
			ui.Print(err)
		}
	}()

	// Make a copy of the AWS SDK config that we're going to use to bounce
	// through user/CredentialFactory in order to get 12-hour credentials so
	// that we don't ruin cfg for whatever else we might want to do with it.
	cfg12h := cfg.Copy()

	callerIdentity, err := cfg12h.SetCredentials(ctx, aws.Credentials{
		AccessKeyID:     aws.ToString(accessKey.AccessKeyId),
		SecretAccessKey: aws.ToString(accessKey.SecretAccessKey),
	})
	if err != nil {
		ui.PrintWithCaller(err)
		return
	}
	cfg12h, err = cfg12h.AssumeRole(
		ctx,
		aws.ToString(callerIdentity.Account),
		roleName,
		12*time.Hour,
	)
	if err != nil {
		ui.PrintWithCaller(err)
		return
	}

	return cfg12h.Retrieve(ctx)
}
