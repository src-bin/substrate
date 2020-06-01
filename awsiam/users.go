package awsiam

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
)

func CreateAccessKey(svc *iam.IAM, username string) (*iam.AccessKey, error) {
	in := &iam.CreateAccessKeyInput{
		UserName: aws.String(username),
	}
	out, err := svc.CreateAccessKey(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.AccessKey, nil
}

func CreateUser(svc *iam.IAM, username string) (*iam.User, error) {
	in := &iam.CreateUserInput{
		Tags:     tagsFor(username),
		UserName: aws.String(username),
	}
	out, err := svc.CreateUser(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.User, nil
}

func DeleteAccessKey(svc *iam.IAM, username, accessKeyId string) error {
	in := &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(accessKeyId),
		UserName:    aws.String(username),
	}
	_, err := svc.DeleteAccessKey(in)
	return err
}

func DeleteAllAccessKeys(svc *iam.IAM, username string) error {
	meta, err := ListAccessKeys(svc, username)
	if err != nil {
		return err
	}
	for _, m := range meta {
		if err := DeleteAccessKey(svc, username, aws.StringValue(m.AccessKeyId)); err != nil {
			return err
		}
	}
	return nil
}

func EnsureUser(svc *iam.IAM, username string) (*iam.User, error) {

	user, err := CreateUser(svc, username)
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		user, err = GetUser(svc, username)
	}
	if err != nil {
		return nil, err
	}

	in := &iam.TagUserInput{
		Tags:     tagsFor(username),
		UserName: aws.String(username),
	}
	if _, err := svc.TagUser(in); err != nil {
		return nil, err
	}

	return user, nil
}

func EnsureUserWithPolicy(svc *iam.IAM, username string, doc *policies.Document) (*iam.User, error) {

	user, err := EnsureUser(svc, username)
	if err != nil {
		return nil, err
	}

	// TODO attach the managed AdministratorAccess policy instead of inlining.
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	in := &iam.PutUserPolicyInput{
		PolicyDocument: aws.String(docJSON),
		PolicyName:     aws.String(SubstrateManaged),
		UserName:       aws.String(username),
	}
	if _, err := svc.PutUserPolicy(in); err != nil {
		return nil, err
	}

	return user, nil
}

func GetUser(svc *iam.IAM, username string) (*iam.User, error) {
	in := &iam.GetUserInput{
		UserName: aws.String(username),
	}
	out, err := svc.GetUser(in)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", out)
	return out.User, nil
}

func ListAccessKeys(svc *iam.IAM, username string) ([]*iam.AccessKeyMetadata, error) {
	in := &iam.ListAccessKeysInput{
		UserName: aws.String(username),
	}
	out, err := svc.ListAccessKeys(in)
	if err != nil {
		return nil, err
	}
	return out.AccessKeyMetadata, err
}
