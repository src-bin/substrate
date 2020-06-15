package awssessions

import (
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

// TODO make a Session type here that lazily constructs and caches all the
// service/*.New clients so we don't have to keep awkwardly bouncing around.

type Config struct {
	AccessKeyId, SecretAccessKey, SessionToken string
	Region                                     string
}

func (c Config) AWS() aws.Config {
	var a aws.Config

	if c.AccessKeyId != "" && c.SecretAccessKey != "" && c.SessionToken != "" {
		a.Credentials = credentials.NewStaticCredentials(c.AccessKeyId, c.SecretAccessKey, c.SessionToken)
	} else if c.AccessKeyId != "" && c.SecretAccessKey != "" {
		a.Credentials = credentials.NewStaticCredentials(c.AccessKeyId, c.SecretAccessKey, "")
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
	return AssumeRole(sess, aws.StringValue(org.MasterAccountId), rolename), nil
}

// InAccount returns a session in the given account (by domain, environment,
// and quality) in the given role or an error if it can't assume that role for
// any reason.  It supports starting from the OrganizationAdministrator role,
// root credentials in the master account, or any role in any account in the
// organization that can assume the given role.
func InAccount(
	domain, environment, quality, rolename string,
	config Config,
) (*session.Session, error) {
	return inAccountByName(awsorgs.NameFor(domain, environment, quality), rolename, config)
}

// InMasterAccount returns a session in the organization's master account in
// the given role or an error if it can't assume the role there for any reason.
// It supports starting from the desired role, root credentials in the master
// account, or any role in any account in the organization that can assume the
// given role.
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

	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	//log.Printf("%+v", org)
	//log.Printf("%+v", err)
	var masterAccountId string
	if awsutil.ErrorCodeIs(err, awsorgs.AWSOrganizationsNotInUseException) {
		err = nil
		masterAccountId = aws.StringValue(callerIdentity.Account)
	} else {
		masterAccountId = aws.StringValue(org.MasterAccountId)
	}
	if err != nil {
		return nil, err
	}

	// Maybe we're already in the desired role.
	if aws.StringValue(callerIdentity.Arn) == roles.Arn(masterAccountId, rolename) {
		return sess, nil
	}

	// Nope.
	sess = AssumeRole(sess, masterAccountId, rolename)

	// Now force it to actually assume the role so that, if we fail, we fail
	// at a sensible time instead of "later."
	callerIdentity, err = awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", callerIdentity)

	return sess, nil
}

// InSpecialAccount returns a session in the given special account (by name)
// in the given role or an error if it can't assume that role for any reason.
// It supports starting from the OrganizationAdministrator role, root
// credentials in the master account, or any role in any account in the
// organization that can assume the given role.
func InSpecialAccount(name, rolename string, config Config) (*session.Session, error) {
	return inAccountByName(name, rolename, config)
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

	// If we're not using root credentials, we're done.
	callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		return nil, err
	}
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

	ui.Stopf("switched to access key %s", accessKey.AccessKeyId)

	return sess, nil
}

func inAccountByName(name, rolename string, config Config) (*session.Session, error) {
	sess, err := NewSession(config)
	if err != nil {
		return nil, err
	}

	masterSess, err := AssumeRoleMaster(sess, roles.OrganizationReader)
	if err != nil {
		return nil, err
	}
	account, err := awsorgs.FindSpecialAccount(organizations.New(masterSess), name)
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", account)

	// Maybe we're already in the desired role.
	callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		return nil, err
	}
	//log.Printf("%+v", callerIdentity)
	if aws.StringValue(callerIdentity.Arn) == roles.Arn(aws.StringValue(account.Id), rolename) {
		return sess, nil
	}

	// Nope.
	return AssumeRole(sess, aws.StringValue(account.Id), rolename), nil
}

func options(config aws.Config) session.Options {
	return session.Options{
		Config:            config,
		SharedConfigState: session.SharedConfigDisable,
	}
}
