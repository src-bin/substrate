package awsiamusers

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	EntityAlreadyExists = "EntityAlreadyExists"

	SubstrateManaged = "SubstrateManaged"
)

type (
	AccessKey         = types.AccessKey
	AccessKeyMetadata = types.AccessKeyMetadata
	Tag               = types.Tag
	User              = types.User
)

func AttachUserPolicy(
	ctx context.Context,
	client *iam.Client,
	username, policyARN string,
) error {
	ui.Spinf("attaching %s to the %s IAM user", policyARN, username)
	_, err := client.AttachUserPolicy(ctx, &iam.AttachUserPolicyInput{
		PolicyArn: aws.String(policyARN),
		UserName:  aws.String(username),
	})

	// We would usually want to suppress a policy-is-already-attached error
	// here but the API doesn't appear to return such an error.

	return ui.StopErr(err)
}

func CreateAccessKey(
	ctx context.Context,
	client *iam.Client,
	username string,
) (*AccessKey, error) {
	out, err := client.CreateAccessKey(ctx, &iam.CreateAccessKeyInput{
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
	client *iam.Client,
	username string,
) (*User, error) {
	out, err := client.CreateUser(ctx, &iam.CreateUserInput{
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

func DeleteAccessKey(
	ctx context.Context,
	client *iam.Client,
	username, accessKeyId string,
) error {
	_, err := client.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(accessKeyId),
		UserName:    aws.String(username),
	})
	return err
}

func DeleteAllAccessKeys(
	ctx context.Context,
	client *iam.Client,
	username string,
	age time.Duration,
) error {
	meta, err := ListAccessKeys(ctx, client, username)
	if err != nil {
		return err
	}
	for _, m := range meta {
		if time.Since(aws.ToTime(m.CreateDate)) > age {
			if err := DeleteAccessKey(ctx, client, username, aws.ToString(m.AccessKeyId)); err != nil {
				return err
			}
		} else if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" { // if we're reasonably sure we're in a Lambda function
			log.Printf(
				"not deleting access key %s because it was created %v, less than %v ago",
				aws.ToString(m.AccessKeyId),
				aws.ToTime(m.CreateDate),
				age,
			)
		}
	}
	return nil
}

func EnsureUser(
	ctx context.Context,
	client *iam.Client,
	username string,
) (*User, error) {
	ui.Spinf("creating the %s IAM user", username)
	user, err := CreateUser(ctx, client, username)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		ui.Stop("already exists")
		ui.Spinf("updating the %s IAM user", username)
		user, err = GetUser(ctx, client, username)
	}
	if err != nil {
		return nil, ui.StopErr(err)
	}

	if _, err := client.TagUser(ctx, &iam.TagUserInput{
		Tags:     tagsFor(username),
		UserName: aws.String(username),
	}); err != nil {
		return nil, ui.StopErr(err)
	}

	ui.Stop("ok")
	return user, nil
}

func EnsureUserWithPolicy(
	ctx context.Context,
	client *iam.Client,
	username string,
	doc *policies.Document,
) (*User, error) {

	user, err := EnsureUser(ctx, client, username)
	if err != nil {
		return nil, err
	}

	// TODO attach the managed AdministratorAccess policy instead of inlining.
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := client.PutUserPolicy(ctx, &iam.PutUserPolicyInput{
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
	client *iam.Client,
	username string,
) (*User, error) {
	out, err := client.GetUser(ctx, &iam.GetUserInput{
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
	client *iam.Client,
	username string,
) ([]AccessKeyMetadata, error) {
	out, err := client.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: aws.String(username),
	})
	if err != nil {
		return nil, err
	}
	return out.AccessKeyMetadata, err
}

func tagsFor(name string) []Tag {
	return []Tag{
		{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
		{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
	}
}
