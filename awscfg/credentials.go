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
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const WaitUntilCredentialsWorkTries = 18 // approximately seconds; API Gateway won't wait longer than 29 seconds, anyway

func (c *Config) BootstrapCredentials(ctx context.Context) (callerIdentity *sts.GetCallerIdentityOutput, err error) {
	callerIdentity, err = c.GetCallerIdentity(ctx)

	// If we already have valid, non-root credentials, cut and run.
	if err == nil && !strings.HasSuffix(aws.ToString(callerIdentity.Arn), ":root") {
		return
	}

	// Gather some credentials interactively, if we need to and are allowed.
	if err != nil {
		if ui.Interactivity() == ui.NonInteractive {
			ui.Fatal("we need AWS credentials but aren't allowed to ask for any; re-run this command with -fully-interactive")
		}
		ui.Print("unable to find any AWS credentials")
		ui.Print("please provide an access key ID and secret access key from either the root of your AWS organization's management account or a user with AdministratorAccess in that same account")
		ui.Print("if you don't have an AWS organization yet, use the account you want to become your AWS organization's management account")
		var creds aws.Credentials
		if creds.AccessKeyID, err = ui.Prompt("AWS access key ID:"); err != nil {
			return
		}
		if creds.SecretAccessKey, err = ui.Prompt("AWS secret access key:"); err != nil {
			return
		}
		ui.Printf("using access key ID %s", creds.AccessKeyID)
		callerIdentity, err = c.SetCredentials(ctx, creds)
	}
	if err != nil {
		return
	}
	accountId := aws.ToString(callerIdentity.Account)

	// Ensure we're either not yet an organization or that these credentials
	// are from its management account.
	var org *Organization
	org, err = c.DescribeOrganization(ctx)
	if err == nil {
		if mgmtAccountId := aws.ToString(org.MasterAccountId); accountId != mgmtAccountId {
			err = NonManagementAccountError{accountId, mgmtAccountId}
			return
		}
	} else {
		if !awsutil.ErrorCodeIs(err, AWSOrganizationsNotInUseException) {
			return
		}
	}

	// Ensure the management account ID matches the file on disk. This is
	// mostly a safety feature for Substrate developers who juggle lots of
	// AWS organizations.
	if err = EnsureManagementAccountIdMatchesDisk(accountId); err != nil {
		return
	}

	// Return early if we have an IAM user credential which will be allowed
	// to assume roles.
	if !strings.HasSuffix(aws.ToString(callerIdentity.Arn), ":root") {
		return
	}
	ui.Spin("switching from root credentials to an IAM user that can assume roles")

	client := c.IAM()

	var user *awsiamusers.User
	user, err = awsiamusers.EnsureUser(ctx, client, users.Substrate)
	if err != nil {
		return
	}
	if err = awsiamusers.AttachUserPolicy(ctx, client, aws.ToString(user.UserName), policies.AdministratorAccess); err != nil {
		return
	}
	//log.Printf("%+v", user)

	if err = awsiamusers.DeleteAllAccessKeys(
		ctx,
		client,
		aws.ToString(user.UserName),
		0,
	); err != nil {
		return
	}

	var accessKey *awsiamusers.AccessKey
	accessKey, err = awsiamusers.CreateAccessKey(
		ctx,
		client,
		aws.ToString(user.UserName),
	)
	if err != nil {
		return
	}
	//log.Printf("%+v", accessKey)
	creds := aws.Credentials{
		AccessKeyID:     aws.ToString(accessKey.AccessKeyId),
		SecretAccessKey: aws.ToString(accessKey.SecretAccessKey),
	}

	// In every other scenario, it's best to leave well enough alone
	// concerning the AWS credentials in this process' (and thus all child
	// processes') environment(s). However, when we accept (root, presumably)
	// credentials on the command line and exchange them for IAM user
	// credentials, nothing in any child process is going to work UNLESS we
	// stuff them into the environment.
	//
	// TODO it's probably time for a rethink of when these are set and what
	// that means for child processes like `terraform plan|apply` that tend
	// today to end up jumping out of the management account during initial
	// setup and from then on out of the Substrate account.
	if err = cmdutil.Setenv(creds); err != nil {
		return
	}

	callerIdentity, err = c.SetCredentials(ctx, creds)
	ui.Stopf("switched to access key ID %s", creds.AccessKeyID)
	return
}

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

	if callerIdentity, err = c.WaitUntilCredentialsWork(ctx); err != nil {
		return
	}

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

// WaitUntilCredentialsWork waits in a sleeping loop until the configured
// credentials (whether provided via SetCredentials or discovered in
// environment variables or an IAM instance profile) work, which it tests
// using both sts:GetCallerIdentity and organizations:DescribeOrganization.
// This seems silly but IAM is an eventually consistent global service so
// it's not guaranteed that newly created credentials will work immediately.
// In fact, even just testing via sts:GetCallerIdentity is demonstrably not
// good enough as `substrate bootstrap-management-account`, when run with
// root credentials, will fail a significant fraction of the time because,
// though sts:GetCallerIdentity succeeded, the credentials haven't yet become
// visible to other services. Thus, organizations:DescribeOrganization was
// chosen as a second test to ensure the credentials really, actually work.
// Typically when this has to wait it waits about five seconds.
func (c *Config) WaitUntilCredentialsWork(ctx context.Context) (
	callerIdentity *sts.GetCallerIdentityOutput,
	err error,
) {
	c.getCallerIdentityOutput = nil // be double sure not to use cached results
	for i := 0; i < WaitUntilCredentialsWorkTries; i++ {
		if callerIdentity, err = c.GetCallerIdentity(ctx); err == nil {
			if _, err = c.DescribeOrganization(ctx); err == nil {
				break
			} else if awsutil.ErrorCodeIs(err, AWSOrganizationsNotInUseException) {
				err = nil
			}
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
