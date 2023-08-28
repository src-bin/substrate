package awsiam

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awssecretsmanager"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	AccessKeyExpiry = 7 * 24 * time.Hour

	CreateAccessKeyTriesBeforeDeleteAll = 4 // must be lower than...
	CreateAccessKeyTriesTotal           = 8 // ...this
)

func AllDayConfig(ctx context.Context, cfg *awscfg.Config) (cfg12h *awscfg.Config, err error) {
	var accessKey *types.AccessKey
	for i := 0; i < CreateAccessKeyTriesTotal; i++ {
		var secret string

		// Look for a cached access key in Secrets Manager. This will save
		// about ten seconds, if we find one.
		if secret, err = awssecretsmanager.GetSecretValue(
			ctx,
			cfg,
			naming.Substrate,
			awssecretsmanager.AWSCURRENT,
		); err == nil {

			// If we found an access key, deserialize it, make sure it's fresh
			// (and delete it if it isn't because auditors hate finding old
			// access keys lying around), and make sure it works.
			if err = json.Unmarshal([]byte(secret), &accessKey); err != nil {
				return
			}
			if time.Since(aws.ToTime(accessKey.CreateDate)) < AccessKeyExpiry {

				// Test the access key and return it if it works.
				if cfg12h, err = allDayConfig(ctx, cfg, accessKey); err == nil {
					log.Printf("using cached access key %s", aws.ToString(accessKey.AccessKeyId))
					return
				} else {
					log.Printf("cached access key %s was probably deleted (%s)", aws.ToString(accessKey.AccessKeyId), err)
				}

			} else {
				log.Printf("deleting cached and expired access key %s", aws.ToString(accessKey.AccessKeyId))
				if err = DeleteAccessKey(
					ctx,
					cfg,
					users.Substrate,
					aws.ToString(accessKey.AccessKeyId),
				); err != nil {
					log.Print(err) // not fatal because a concurrent actor may have deleted this one
				}
			}

		}

		// If we didn't find an access key in Secrets Manager, try pretty
		// hard to create one, backing off and trying to play nice within
		// the two-access-keys-per-user limit and the potential for
		// competition with others using the Credential Factory.
		accessKey, err = CreateAccessKey(ctx, cfg, users.Substrate)
		if awsutil.ErrorCodeIs(err, LimitExceeded) {
			if i == CreateAccessKeyTriesBeforeDeleteAll {
				if err = DeleteAllAccessKeys(
					ctx,
					cfg,
					users.Substrate,
					time.Minute, // don't delete access keys that were just created; they might be cached next time around
				); err != nil {
					return
				}
			}
			continue
		} else if err != nil {
			return
		}

		// Cache the access key we just created in Secrets Manager.
		log.Printf("caching access key %s", aws.ToString(accessKey.AccessKeyId))
		if secret, err = jsonutil.OneLineString(accessKey); err == nil {
			if _, err := awssecretsmanager.EnsureSecret(
				ctx,
				cfg,
				naming.Substrate,
				awssecretsmanager.Policy(&policies.Principal{AWS: []string{
					roles.ARN(cfg.MustAccountId(ctx), roles.Intranet),
				}}),
				awssecretsmanager.AWSCURRENT,
				secret,
			); err != nil {
				ui.PrintWithCaller(err)
			}
		} else {
			ui.PrintWithCaller(err)
		}

		// Return the access key if it works. There's a very, very slim
		// chance it already won't in case of very high concurrency. In
		// such cases, we expect to go around again and hit in the cache.
		if cfg12h, err = allDayConfig(ctx, cfg, accessKey); err == nil {
			return
		} else {
			ui.PrintWithCaller(err)
		}

	}
	return
}

func AllDayCredentials(
	ctx context.Context,
	cfg *awscfg.Config,
	accountId, roleName string,
) (creds aws.Credentials, err error) {
	var cfg12h *awscfg.Config

	if cfg12h, err = AllDayConfig(ctx, cfg); err != nil {
		return
	}

	if cfg12h, err = cfg12h.AssumeRole(
		ctx,
		accountId,
		roleName,
		12*time.Hour,
	); err != nil {
		return
	}

	return cfg12h.Retrieve(ctx)
}

func allDayConfig(ctx context.Context, cfg *awscfg.Config, accessKey *types.AccessKey) (cfg12h *awscfg.Config, err error) {

	// Make a copy of the AWS SDK config that we're going to use to bounce
	// through user/Substrate in order to get 12-hour credentials so that we
	// don't ruin cfg for whatever else we might want to do with it.
	cfg12h = cfg.Copy()

	_, err = cfg12h.SetCredentials(ctx, aws.Credentials{
		AccessKeyID:     aws.ToString(accessKey.AccessKeyId),
		SecretAccessKey: aws.ToString(accessKey.SecretAccessKey),
	})
	if err != nil {
		cfg12h = nil
	}

	return
}
