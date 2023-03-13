package awssecretsmanager

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

const (
	AWSCURRENT              = "" // the default when no stage is given is AWSCURRENT
	ResourceExistsException = "ResourceExistsException"
)

type (
	CreateSecretOutput      = secretsmanager.CreateSecretOutput
	DescribeSecretOutput    = secretsmanager.DescribeSecretOutput
	PutResourcePolicyOutput = secretsmanager.PutResourcePolicyOutput
	PutSecretValueOutput    = secretsmanager.PutSecretValueOutput
)

func CreateSecret(ctx context.Context, cfg *awscfg.Config, name string) (*CreateSecretOutput, error) {
	out, err := cfg.SecretsManager().CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name: aws.String(name),
		Tags: []types.Tag{
			{
				Key:   aws.String(tagging.Manager),
				Value: aws.String(tagging.Substrate),
			},
			{
				Key:   aws.String(tagging.SubstrateVersion),
				Value: aws.String(version.Version),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func DescribeSecret(ctx context.Context, cfg *awscfg.Config, name string) (*DescribeSecretOutput, error) {
	out, err := cfg.SecretsManager().DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func EnsureSecret(
	ctx context.Context,
	cfg *awscfg.Config,
	name string,
	doc *policies.Document,
	stage, value string,
) (*PutSecretValueOutput, error) {

	_, err := CreateSecret(ctx, cfg, name)
	if awsutil.ErrorCodeIs(err, ResourceExistsException) {
		err = nil
	}
	if err != nil {
		return nil, err
	}

	if _, err := PutResourcePolicy(ctx, cfg, name, doc); err != nil {
		return nil, err
	}

	return PutSecretValue(ctx, cfg, name, stage, value)
}

func GetSecretValue(ctx context.Context, cfg *awscfg.Config, name, stage string) (string, error) {
	in := &secretsmanager.GetSecretValueInput{SecretId: aws.String(name)}
	if stage != "" {
		in.VersionStage = aws.String(stage)
	}
	out, err := cfg.SecretsManager().GetSecretValue(ctx, in)
	if err != nil {
		return "", err
	}
	return aws.ToString(out.SecretString), nil
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

func PutResourcePolicy(ctx context.Context, cfg *awscfg.Config, name string, doc *policies.Document) (*PutResourcePolicyOutput, error) {
	docJSON, err := doc.Marshal()
	if err != nil {
		return nil, err
	}
	out, err := cfg.SecretsManager().PutResourcePolicy(ctx, &secretsmanager.PutResourcePolicyInput{
		ResourcePolicy: aws.String(docJSON),
		SecretId:       aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func PutSecretValue(ctx context.Context, cfg *awscfg.Config, name, stage, value string) (*PutSecretValueOutput, error) {
	var stages []string
	if stage != "" {
		stages = []string{stage}
	}
	out, err := cfg.SecretsManager().PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:      aws.String(name),
		SecretString:  aws.String(value),
		VersionStages: stages,
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
