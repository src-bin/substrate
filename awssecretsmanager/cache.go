package awssecretsmanager

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
)

var secrets = &sync.Map{}

func CachedSecret(ctx context.Context, cfg *awscfg.Config, name, stage string) (string, error) {
	if v, ok := secrets.Load(name); ok {
		return v.(string), nil
	}

	out, err := GetSecretValue(ctx, cfg, name, stage)
	if err != nil {
		return "", err
	}
	secret := aws.ToString(out.SecretString)

	secrets.Store(name, secret)
	return secret, nil
}
