package awssessions

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	NewSessionTries       = 10
	NoCredentialProviders = "NoCredentialProviders"
)

// TODO make a Session type here that lazily constructs and caches all the
// service/*.New clients so we don't have to keep awkwardly bouncing around.

type Config struct {
	AccessKeyId, SecretAccessKey, SessionToken string
	Region                                     string
}

func (c Config) AWS() aws.Config {
	var a aws.Config

	if c.AccessKeyId != "" && c.SecretAccessKey != "" {
		a.Credentials = credentials.NewStaticCredentials(c.AccessKeyId, c.SecretAccessKey, c.SessionToken)
	}
	if c.AccessKeyId != "" && c.SecretAccessKey == "" {
		ui.Print("ignoring access key ID without secret access key")
	}
	if c.AccessKeyId == "" && c.SecretAccessKey != "" {
		ui.Print("ignoring secret access key without access key ID")
	}

	if c.Region != "" {
		a.Region = aws.String(c.Region)
	}

	return a
}

// TODO AssumeRoleArn(sess, roleArn) variant?
func AssumeRole(sess *session.Session, accountId, rolename string) *session.Session {
	arn := roles.Arn(accountId, rolename)
	ui.Printf("assuming role %s", arn)
	return sess.Copy(&aws.Config{Credentials: stscreds.NewCredentials(sess, arn)})
}

func AssumeRoleMaster(sess *session.Session, rolename string) (*session.Session, error) {
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if err != nil {
		return nil, err
	}
	if err := accounts.EnsureMasterAccountIdMatchesDisk(aws.StringValue(org.MasterAccountId)); err != nil {
		return nil, err
	}
	return AssumeRole(sess, aws.StringValue(org.MasterAccountId), rolename), nil
}

// InAccount returns a session in the given account (by domain, environment,
// and quality) in the given role or an error if it can't assume that role for
// any reason.  It supports starting from the OrganizationAdministrator role,
// root credentials in the master account, or any role in any account in the
// organization that can assume the given role.
//
// The initial identity assumed first before assuming a role in the other
// account must be allowed to call organizations:DescribeOrganization and
// sts:AssumeRole.
func InAccount(
	domain, environment, quality, rolename string,
	config Config,
) (*session.Session, error) {
	return InSpecialAccount(awsorgs.NameFor(domain, environment, quality), rolename, config)
}

// InMasterAccount returns a session in the organization's master account in
// the given role or an error if it can't assume the role there for any reason.
// It supports starting from the desired role, root credentials in the master
// account, or any role in any account in the organization that can assume the
// given role.
//
// The initial identity assumed first before assuming a role in the master
// account must be allowed to call organizations:DescribeOrganization and
// sts:AssumeRole.
func InMasterAccount(rolename string, config Config) (*session.Session, error) {
	sess, err := NewSession(config)
	if err != nil {
		return nil, err
	}
	callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", callerIdentity)

	// Figure out the master account ID.  If there isn't even an organization
	// yet, it's this account's ID.
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if awsutil.ErrorCodeIs(err, awsorgs.AWSOrganizationsNotInUseException) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	var masterAccountId string
	if org == nil {
		masterAccountId = aws.StringValue(callerIdentity.Account)
	} else {
		masterAccountId = aws.StringValue(org.MasterAccountId)
	}
	if err := accounts.EnsureMasterAccountIdMatchesDisk(masterAccountId); err != nil {
		return nil, err
	}
	callerIdentityArn := aws.StringValue(callerIdentity.Arn)

	// Maybe we're already in the desired role.
	if callerIdentityArn == roles.Arn(masterAccountId, rolename) {
		return sess, nil
	}

	// Or maybe we're trying to be role/OrganizationAdministrator but really
	// user/OrganizationAdministrator will do.
	if rolename == roles.OrganizationAdministrator && callerIdentityArn == users.Arn(masterAccountId, users.OrganizationAdministrator) {
		return sess, nil
	}

	// Nope.
	sess = AssumeRole(sess, masterAccountId, rolename)

	// Now force it to actually assume the role so that, if we fail, we fail
	// at a sensible time instead of "later."
	if _, err := awssts.GetCallerIdentity(sts.New(sess)); err != nil {

		// Offer one (and only one) more shot via root credentials.
		if config.AccessKeyId == "" {
			return InMasterAccount(rolename, configWithRootCredentials(rolename, config))
		}

		return nil, err
	}

	return sess, nil
}

// InSpecialAccount returns a session in the given special account (by name)
// in the given role or an error if it can't assume that role for any reason.
// It supports starting from the OrganizationAdministrator role, root
// credentials in the master account, or any role in any account in the
// organization that can assume the given role.
//
// The initial identity assumed first before assuming a role in the other
// account must be allowed to call organizations:DescribeOrganization and
// sts:AssumeRole.
func InSpecialAccount(name, rolename string, config Config) (*session.Session, error) {
	sess, err := NewSession(config)
	if err != nil {
		return nil, err
	}

	masterSess, err := AssumeRoleMaster(sess, roles.OrganizationReader)
	if err != nil {

		// But if we never even got started, and we haven't already asked, ask
		// for root credentials and try again.
		if config.AccessKeyId == "" {
			return InSpecialAccount(name, rolename, configWithRootCredentials(rolename, config))
		}

		return nil, err
	}
	account, err := awsorgs.FindSpecialAccount(organizations.New(masterSess), name)
	if err != nil {
		return nil, err
	}

	// Maybe we're already in the desired role.
	callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		return nil, err
	}
	if aws.StringValue(callerIdentity.Arn) == roles.Arn(aws.StringValue(account.Id), rolename) {
		return sess, nil
	}

	// Nope.
	sess = AssumeRole(sess, aws.StringValue(account.Id), rolename)

	// Now force it to actually assume the role so that, if we fail, we fail
	// at a sensible time instead of "later."
	if _, err := awssts.GetCallerIdentity(sts.New(sess)); err != nil {

		// Offer one (and only one) more shot via root credentials.
		if config.AccessKeyId == "" {
			return InSpecialAccount(name, rolename, configWithRootCredentials(rolename, config))
		}

		return nil, err
	}

	return sess, nil
}

func Must(sess *session.Session, err error) *session.Session {
	if err != nil {
		ui.Fatal(err)
	}
	return sess
}

// NewSession constucts an AWS session from whatever given and environmental
// configuration it can find.  If it's given root credentials then it creates
// an IAM user and an access key so that sts:AssumeRole will be available.
func NewSession(config Config) (*session.Session, error) {
	sess, err := session.NewSessionWithOptions(options(config.AWS()))

	// Take a bounded amount of time to let newly-minted credentials become
	// valid but don't spin forever because it may genuinely be the case that
	// the credentials are invalid.
	var callerIdentity *sts.GetCallerIdentityOutput
	for i := 0; i < NewSessionTries; i++ {
		callerIdentity, err = awssts.GetCallerIdentity(sts.New(sess))
		if awsutil.ErrorCodeIs(err, awssts.InvalidClientTokenId) {
			time.Sleep(1e9) // TODO exponential backoff
			continue
		}
		if awsutil.ErrorCodeIs(err, NoCredentialProviders) {

			// In this case the AWS SDK couldn't find any credentials so let's
			// ask for some and try again.
			if config.AccessKeyId == "" {
				return NewSession(configWithRootCredentials("", config))
			}

		}

		break
	}
	if err != nil {
		return nil, err
	}

	// If we're not using root credentials, we're done.
	if !strings.HasSuffix(aws.StringValue(callerIdentity.Arn), ":root") {
		ui.Printf("starting AWS session as %s", callerIdentity.Arn)
		return sess, nil
	}

	ui.Spin("switching from root credentials to an IAM user that can assume roles")
	svc := iam.New(sess)

	user, err := awsiam.EnsureUserWithPolicy(
		svc,
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

	if err := awsiam.DeleteAllAccessKeys(
		svc,
		users.OrganizationAdministrator,
	); err != nil {
		return nil, err
	}

	accessKey, err := awsiam.CreateAccessKey(svc, aws.StringValue(user.UserName))
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", accessKey)
	/*
		defer awsiam.DeleteAllAccessKeys(
			svc,
			users.OrganizationAdministrator,
		) // TODO ensure this succeeds even when we exit via log.Fatal
	*/

	sess, err = session.NewSessionWithOptions(options(Config{
		AccessKeyId:     aws.StringValue(accessKey.AccessKeyId),
		SecretAccessKey: aws.StringValue(accessKey.SecretAccessKey),
		Region:          config.Region,
	}.AWS()))
	if err != nil {
		return nil, err
	}

	// Override the environment and any discovered credentials for all child
	// processes, which smooths out Terraform runs initiated by this process.
	// TODO factor this out and also do it in configWithRootCredentials
	if err := os.Setenv("AWS_ACCESS_KEY_ID", aws.StringValue(accessKey.AccessKeyId)); err != nil {
		return nil, err
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", aws.StringValue(accessKey.SecretAccessKey)); err != nil {
		return nil, err
	}
	if err := os.Unsetenv("AWS_SESSION_TOKEN"); err != nil {
		return nil, err
	}

	// Inconceivably, the new access key probably isn't usable for a
	// little while so we have to sit and spin before using it.
	for {
		_, err := awssts.GetCallerIdentity(sts.New(sess))
		if err == nil {
			break
		}
		if !awsutil.ErrorCodeIs(err, awssts.InvalidClientTokenId) {
			return nil, err
		}
		time.Sleep(1e9) // TODO exponential backoff
	}
	time.Sleep(5e9) // even when the loop above ends, we still might have to wait

	ui.Stopf("switched to access key ID %s", accessKey.AccessKeyId)

	return sess, nil
}

func configWithRootCredentials(rolename string, config Config) Config {
	if rolename == "" {
		ui.Printf(
			"unable to find any AWS credentials, which means this is probably your first time running %s",
			filepath.Base(os.Args[0]),
		)
	} else {
		ui.Printf(
			"unable to assume the %s role, which means this is probably your first time running %s",
			rolename,
			filepath.Base(os.Args[0]),
		)
	}
	ui.Print("please provide an access key from your master AWS account")
	config.AccessKeyId, config.SecretAccessKey = awsutil.ReadAccessKeyFromStdin()
	config.SessionToken = ""
	ui.Printf("using access key ID %s", config.AccessKeyId)
	return config
}

func options(config aws.Config) session.Options {
	return session.Options{
		Config:            config,
		SharedConfigState: session.SharedConfigDisable,
	}
}
