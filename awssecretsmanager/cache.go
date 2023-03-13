package awssecretsmanager

import (
	"context"
	"sync"

	"github.com/src-bin/substrate/awscfg"
)

var secrets = &sync.Map{}

func CachedSecret(ctx context.Context, cfg *awscfg.Config, name, stage string) (string, error) {
	if v, ok := secrets.Load(name); ok {
		return v.(string), nil
	}

	secret, err := GetSecretValue(ctx, cfg, name, stage)
	if err != nil {
		return "", err
	}

	secrets.Store(name, secret)
	return secret, nil
}
