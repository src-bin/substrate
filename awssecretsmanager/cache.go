package awssecretsmanager

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var secrets = &sync.Map{}

func CachedSecret(svc *secretsmanager.SecretsManager, name, stage string) (string, error) {
	if v, ok := secrets.Load(name); ok {
		return v.(string), nil
	}

	out, err := GetSecretValue(svc, name, stage)
	if err != nil {
		return "", err
	}
	secret := aws.StringValue(out.SecretString)

	secrets.Store(name, secret)
	return secret, nil
}
