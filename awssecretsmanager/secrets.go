package awssecretsmanager

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/version"
)

const ResourceExistsException = "ResourceExistsException"

func CreateSecret(svc *secretsmanager.SecretsManager, name string) (*secretsmanager.CreateSecretOutput, error) {
	in := &secretsmanager.CreateSecretInput{
		Name: aws.String(name),
		Tags: []*secretsmanager.Tag{
			&secretsmanager.Tag{
				Key:   aws.String(tags.Manager),
				Value: aws.String(tags.Substrate),
			},
			&secretsmanager.Tag{
				Key:   aws.String(tags.SubstrateVersion),
				Value: aws.String(version.Version),
			},
		},
	}
	out, err := svc.CreateSecret(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func DescribeSecret(svc *secretsmanager.SecretsManager, name string) (*secretsmanager.DescribeSecretOutput, error) {
	in := &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	}
	out, err := svc.DescribeSecret(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func EnsureSecret(
	svc *secretsmanager.SecretsManager,
	name string,
	doc *policies.Document,
	stage, value string,
) (*secretsmanager.PutSecretValueOutput, error) {

	_, err := CreateSecret(svc, name)
	if awsutil.ErrorCodeIs(err, ResourceExistsException) {
		err = nil
	}
	if err != nil {
		return nil, err
	}

	if _, err := PutResourcePolicy(svc, name, doc); err != nil {
		return nil, err
	}

	return PutSecretValue(svc, name, stage, value)
}

func GetSecretValue(svc *secretsmanager.SecretsManager, name, stage string) (*secretsmanager.GetSecretValueOutput, error) {
	in := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(name),
		VersionStage: aws.String(stage),
	}
	out, err := svc.GetSecretValue(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func Policy(principal *policies.Principal) *policies.Document {
	return &policies.Document{
		Statement: []policies.Statement{
			policies.Statement{
				Action:    []string{"secretsmanager:GetSecretValue"},
				Principal: principal,
				Resource:  []string{"*"},
			},
		},
	}
}

func PutResourcePolicy(svc *secretsmanager.SecretsManager, name string, doc *policies.Document) (*secretsmanager.PutResourcePolicyOutput, error) {
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	in := &secretsmanager.PutResourcePolicyInput{
		ResourcePolicy: aws.String(docJSON),
		SecretId:       aws.String(name),
	}
	out, err := svc.PutResourcePolicy(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func PutSecretValue(svc *secretsmanager.SecretsManager, name, stage, value string) (*secretsmanager.PutSecretValueOutput, error) {
	in := &secretsmanager.PutSecretValueInput{
		SecretId:      aws.String(name),
		SecretString:  aws.String(value),
		VersionStages: []*string{aws.String(stage)},
	}
	out, err := svc.PutSecretValue(in)
	if err != nil {
		return nil, err
	}
	return out, nil
}
