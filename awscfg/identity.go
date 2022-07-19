package awscfg

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/tags"
)

type Identity struct {
	ARN       string
	AccountID string
	Tags      struct {
		Domain, Environment, Quality string
	}
}

func (c *Config) Identity(ctx context.Context) (*Identity, error) {
	callerIdentity, err := c.GetCallerIdentity(ctx)

	cfg, err := c.OrganizationReader(ctx)
	if err != nil {
		return nil, err
	}
	/*
		a, err := cfg.Organizations().DescribeAccount(
			ctx,
			&organizations.DescribeAccountInput{
				AccountId: callerIdentity.Account,
			},
		)
		if err != nil {
			return nil, err
		}
	*/
	t, err := cfg.listTagsForResource(ctx, aws.ToString(callerIdentity.Account))
	if err != nil {
		return nil, err
	}

	return &Identity{
		ARN:       aws.ToString(callerIdentity.Arn),
		AccountID: aws.ToString(callerIdentity.Account),
		Tags: struct{ Domain, Environment, Quality string }{
			Domain:      t[tags.Domain],
			Environment: t[tags.Environment],
			Quality:     t[tags.Quality],
		},
	}, nil
}
