package awscfg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/src-bin/substrate/awsiam/awsiamusers"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	AWSOrganizationsNotInUseException = "AWSOrganizationsNotInUseException"

	WaitUntilCredentialsWorkTries = 60 // try for one minute (plus request latency)
)

func (c *Config) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return c.cfg.Credentials.Retrieve(ctx)
}

// SetCredentials reconfigures the receiver to use the given credentials
// (whether root, user, or session credentials) and waits until they begin
// working (which concerns mostly user credentials). It returns the caller
// identity because it's already gone to the trouble of getting it and
// callers often need it right afterward, anyway.
func (c *Config) SetCredentials(
	ctx context.Context,
	creds aws.Credentials,
) (
	callerIdentity *sts.GetCallerIdentityOutput,
	err error,
) {
	if c.cfg, err = config.LoadDefaultConfig(
		ctx,
		loadOptions(config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{creds},
		))...,
	); err != nil {
		return
	}

	/*
		// Definitely don't set environment variables when we're in Lambda or
		// we'll ruin the entire future of this process. And, since the purpose
		// of setting these in the first place is to facilitate subprocesses
		// like Terraform, it's blessedly unnecessary in Lambda, anyway.
		if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
			if err = os.Setenv("AWS_ACCESS_KEY_ID", creds.AccessKeyID); err != nil {
				return
			}
			if err = os.Setenv("AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey); err != nil {
				return
			}
			if creds.SessionToken == "" {
				err = os.Unsetenv("AWS_SESSION_TOKEN")
			} else {
				err = os.Setenv("AWS_SESSION_TOKEN", creds.SessionToken)
			}
		}
	*/

	callerIdentity, err = c.WaitUntilCredentialsWork(ctx)

	if c.deferredTelemetry != nil {
		ctx10s, _ := context.WithTimeout(ctx, 10*time.Second)
		if err := c.deferredTelemetry(ctx10s); err == nil {
			c.deferredTelemetry = nil
		} else {
			//log.Print(err)
		}
	}

	return
}

func (c *Config) SetRootCredentials(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {

	// Gather some credentials interactively, if we're allowed.
	if ui.Interactivity() == ui.NonInteractive {
		ui.Fatal("we need AWS credentials but aren't allowed to ask for any; re-run this command with -fully-interactive")
	}
	ui.Print("unable to find any AWS credentials; please provide an access key ID and secret access key from either the root of your AWS management account or the OrganizationAdministrator user in that same account")
	var (
		creds aws.Credentials
		err   error
	)
	if creds.AccessKeyID, err = ui.Prompt("AWS access key ID:"); err != nil {
		ui.Fatal(err)
	}
	if creds.SecretAccessKey, err = ui.Prompt("AWS secret access key:"); err != nil {
		ui.Fatal(err)
	}
	ui.Printf("using access key ID %s", creds.AccessKeyID)
	out, err := c.SetCredentials(ctx, creds)
	accountId := aws.ToString(out.Account)

	// Ensure we're either not yet an organization or that these credentials
	// are from its management account.
	org, err := c.DescribeOrganization(ctx)
	if err == nil {
		if mgmtAccountId := aws.ToString(org.MasterAccountId); accountId != mgmtAccountId {
			return nil, NonManagementAccountError{accountId, mgmtAccountId}
		}
	} else {
		if !awsutil.ErrorCodeIs(err, AWSOrganizationsNotInUseException) {
			return nil, err
		}
	}

	// Ensure the management account ID matches the file on disk. This is
	// mostly a safety feature for Substrate developers who juggle lots of
	// AWS organizations.
	if err := EnsureManagementAccountIdMatchesDisk(accountId); err != nil {
		return nil, err
	}

	// Return early if we have an IAM user credential which will be allowed
	// to assume roles.
	if !strings.HasSuffix(aws.ToString(out.Arn), ":root") {
		return nil, err
	}
	ui.Spin("switching from root credentials to an IAM user that can assume roles")

	client := c.IAM()

	user, err := awsiamusers.EnsureUserWithPolicy(
		ctx,
		client,
		users.OrganizationAdministrator,
		&policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Action:   []string{"*"},
					Resource: []string{"*"},
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", user)

	if err := awsiamusers.DeleteAllAccessKeys(
		ctx,
		client,
		aws.ToString(user.UserName),
	); err != nil {
		return nil, err
	}

	accessKey, err := awsiamusers.CreateAccessKey(
		ctx,
		client,
		aws.ToString(user.UserName),
	)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", accessKey)
	creds = aws.Credentials{
		AccessKeyID:     aws.ToString(accessKey.AccessKeyId),
		SecretAccessKey: aws.ToString(accessKey.SecretAccessKey),
	}

	out, err = c.SetCredentials(ctx, creds)
	ui.Stopf("switched to access key ID %s", creds.AccessKeyID)
	return out, err
}

// WaitUntilCredentialsWork waits in a sleeping loop until the configured
// credentials (whether provided via SetCredentials or discovered in
// environment variables or an IAM instance profile) work, which it tests
// using sts:GetCallerIdentity. This seems silly but IAM is an eventually
// consistent global service so it's not guaranteed that newly created
// credentials will work immediately. Typically when this has to wait it
// waits about five seconds.
func (c *Config) WaitUntilCredentialsWork(ctx context.Context) (
	callerIdentity *sts.GetCallerIdentityOutput,
	err error,
) {
	c.getCallerIdentityOutput = nil // be double sure not to use cached results
	for i := 0; i < WaitUntilCredentialsWorkTries; i++ {
		if callerIdentity, err = c.GetCallerIdentity(ctx); err == nil {
			break
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	return
}

type NonManagementAccountError struct {
	accountId, mgmtAccountId string
}

func (err NonManagementAccountError) Error() string {
	return fmt.Sprintf(
		"credentials are for account %s, not the organization's management account, %s",
		err.accountId,
		err.mgmtAccountId,
	)
}
