package awssecretsmanager

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

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
