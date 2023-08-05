package awsiam

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/version"
)

const (
	GitHubActionsOAuthOIDCThumbprint = "ffffffffffffffffffffffffffffffffffffffff" // <https://github.com/aws-actions/configure-aws-credentials/issues/357>
	GitHubActionsOAuthOIDCURL        = "https://token.actions.githubusercontent.com"
)

func EnsureOpenIDConnectProvider(
	ctx context.Context,
	cfg *awscfg.Config,
	clients, thumbprints []string,
	urlString string, // this is the "primary key"
) (string, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}
	out, err := cfg.IAM().CreateOpenIDConnectProvider(ctx, &iam.CreateOpenIDConnectProviderInput{
		ClientIDList: clients,
		Tags: []types.Tag{
			{Key: aws.String(tagging.Manager), Value: aws.String(tagging.Substrate)},
			{Key: aws.String(tagging.SubstrateVersion), Value: aws.String(version.Version)},
		},
		ThumbprintList: thumbprints,
		Url:            aws.String(u.String()),
	})
	if awsutil.ErrorCodeIs(err, EntityAlreadyExists) {
		callerIdentity, err := cfg.GetCallerIdentity(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(
			"arn:aws:iam::%s:oidc-provider/%s",
			aws.ToString(callerIdentity.Account),
			u.Host,
		), nil
	} else if err != nil {
		return "", err
	}
	return aws.ToString(out.OpenIDConnectProviderArn), nil
}
