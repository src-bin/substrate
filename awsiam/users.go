package awsiam

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam/awsiamusers"
	"github.com/src-bin/substrate/policies"
)

type (
	AccessKey         = types.AccessKey
	AccessKeyMetadata = types.AccessKeyMetadata
	User              = types.User
)

func CreateAccessKey(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*AccessKey, error) {
	return awsiamusers.CreateAccessKey(ctx, cfg.IAM(), username)
}

func CreateUser(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*User, error) {
	return awsiamusers.CreateUser(ctx, cfg.IAM(), username)
}

func DeleteAccessKey(
	ctx context.Context,
	cfg *awscfg.Config,
	username, accessKeyId string,
) error {
	return awsiamusers.DeleteAccessKey(ctx, cfg.IAM(), username, accessKeyId)
}

func DeleteAllAccessKeys(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) error {
	return awsiamusers.DeleteAllAccessKeys(ctx, cfg.IAM(), username)
}

func EnsureUser(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*User, error) {
	return awsiamusers.EnsureUser(ctx, cfg.IAM(), username)
}

func EnsureUserWithPolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
	doc *policies.Document,
) (*User, error) {
	return awsiamusers.EnsureUserWithPolicy(ctx, cfg.IAM(), username, doc)
}

func GetUser(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*User, error) {
	return awsiamusers.GetUser(ctx, cfg.IAM(), username)
}

func ListAccessKeys(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) ([]AccessKeyMetadata, error) {
	return awsiamusers.ListAccessKeys(ctx, cfg.IAM(), username)
}
