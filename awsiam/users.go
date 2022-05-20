package awsiam

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	iamv1 "github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
)

func CreateAccessKey(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*types.AccessKey, error) {
	out, err := cfg.IAM().CreateAccessKey(ctx, &iam.CreateAccessKeyInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.AccessKey, nil
}

func CreateAccessKeyV1(
	svc *iamv1.IAM,
	username string,
) (*iamv1.AccessKey, error) {
	out, err := svc.CreateAccessKey(&iamv1.CreateAccessKeyInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.AccessKey, nil
}

func CreateUser(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*types.User, error) {
	out, err := cfg.IAM().CreateUser(ctx, &iam.CreateUserInput{
		Tags:     tagsFor(username),
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	time.Sleep(10e9) // give IAM time to become consistent (TODO do it gracefully)
	return out.User, nil
}

func CreateUserV1(
	svc *iamv1.IAM,
	username string,
) (*iamv1.User, error) {
	out, err := svc.CreateUser(&iamv1.CreateUserInput{
		Tags:     tagsForV1(username),
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	time.Sleep(10e9) // give IAM time to become consistent (TODO do it gracefully)
	return out.User, nil
}

func DeleteAccessKey(
	ctx context.Context,
	cfg *awscfg.Config,
	username, accessKeyId string,
) error {
	_, err := cfg.IAM().DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(accessKeyId),
		UserName:    aws.String(username),
	})
	return err
}

func DeleteAccessKeyV1(
	svc *iamv1.IAM,
	username, accessKeyId string,
) error {
	_, err := svc.DeleteAccessKey(&iamv1.DeleteAccessKeyInput{
		AccessKeyId: aws.String(accessKeyId),
		UserName:    aws.String(username),
	})
	return err
}

func DeleteAllAccessKeys(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) error {
	meta, err := ListAccessKeys(ctx, cfg, username)
	if err != nil {
		return err
	}
	for _, m := range meta {
		if err := DeleteAccessKey(ctx, cfg, username, aws.ToString(m.AccessKeyId)); err != nil {
			return err
		}
	}
	return nil
}

func DeleteAllAccessKeysV1(
	svc *iamv1.IAM,
	username string,
) error {
	meta, err := ListAccessKeysV1(svc, username)
	if err != nil {
		return err
	}
	for _, m := range meta {
		if err := DeleteAccessKeyV1(svc, username, aws.ToString(m.AccessKeyId)); err != nil {
			return err
		}
	}
	return nil
}

func EnsureUser(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*types.User, error) {

	user, err := CreateUser(ctx, cfg, username)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		user, err = GetUser(ctx, cfg, username)
	}
	if err != nil {
		return nil, err
	}

	if _, err := cfg.IAM().TagUser(ctx, &iam.TagUserInput{
		Tags:     tagsFor(username),
		UserName: aws.String(username),
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func EnsureUserV1(
	svc *iamv1.IAM,
	username string,
) (*iamv1.User, error) {

	user, err := CreateUserV1(svc, username)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		user, err = GetUserV1(svc, username)
	}
	if err != nil {
		return nil, err
	}

	if _, err := svc.TagUser(&iamv1.TagUserInput{
		Tags:     tagsForV1(username),
		UserName: aws.String(username),
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func EnsureUserWithPolicy(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
	doc *policies.Document,
) (*types.User, error) {

	user, err := EnsureUser(ctx, cfg, username)
	if err != nil {
		return nil, err
	}

	// TODO attach the managed AdministratorAccess policy instead of inlining.
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := cfg.IAM().PutUserPolicy(ctx, &iam.PutUserPolicyInput{
		PolicyDocument: aws.String(docJSON),
		PolicyName:     aws.String(SubstrateManaged),
		UserName:       aws.String(username),
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func EnsureUserWithPolicyV1(
	svc *iamv1.IAM,
	username string,
	doc *policies.Document,
) (*iamv1.User, error) {

	user, err := EnsureUserV1(svc, username)
	if err != nil {
		return nil, err
	}

	// TODO attach the managed AdministratorAccess policy instead of inlining.
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := svc.PutUserPolicy(&iamv1.PutUserPolicyInput{
		PolicyDocument: aws.String(docJSON),
		PolicyName:     aws.String(SubstrateManaged),
		UserName:       aws.String(username),
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func GetUser(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) (*types.User, error) {
	out, err := cfg.IAM().GetUser(ctx, &iam.GetUserInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.User, nil
}

func GetUserV1(
	svc *iamv1.IAM,
	username string,
) (*iamv1.User, error) {
	out, err := svc.GetUser(&iamv1.GetUserInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.User, nil
}

func ListAccessKeys(
	ctx context.Context,
	cfg *awscfg.Config,
	username string,
) ([]types.AccessKeyMetadata, error) {
	out, err := cfg.IAM().ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	return out.AccessKeyMetadata, err
}

func ListAccessKeysV1(
	svc *iamv1.IAM,
	username string,
) ([]*iamv1.AccessKeyMetadata, error) {
	out, err := svc.ListAccessKeys(&iamv1.ListAccessKeysInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	return out.AccessKeyMetadata, err
}
