package main

import (
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	Google = "Google"
	Okta   = "Okta"
)

func idp(sess *session.Session, account *awsorgs.Account, metadata string) (name string) {
	svc := iam.New(sess, &aws.Config{
		Credentials: stscreds.NewCredentials(sess, roles.Arn(
			aws.StringValue(account.Id),
			roles.OrganizationAccountAccessRole,
		)),
	})

	name = "IdP"
	if strings.Contains(metadata, "google.com") {
		name = Google
	} else if strings.Contains(metadata, "okta.com") {
		name = Okta
	}

	ui.Spinf("configuring %s as your organization's identity provider", name)
	saml, err := awsiam.EnsureSAMLProvider(svc, name, metadata)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("provider %s", saml.Arn)
	//log.Printf("%+v", saml)

	// Pre-create this user so that it may be referenced in policies attached to
	// the Administrator user.  Terraform will attach policies to it later.
	ui.Spin("finding or creating an IAM user for your Credential Factory, so it can get 12-hour credentials")
	user, err := awsiam.EnsureUser(svc, users.CredentialFactory)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(5e9) // TODO wait only just long enough for IAM to become consistent, and probably do it in EnsureUser
	ui.Stopf("user %s", user.UserName)

	// Give Okta some entrypoints in the Admin account.
	ui.Spinf("finding or creating roles for %s to use in this admin account", name)
	adminPrincipals, err := admin.AdminPrincipals(organizations.New(sess))
	if err != nil {
		log.Fatal(err)
	}
	assumeRolePolicyDocument := &policies.Document{
		Statement: []policies.Statement{
			policies.AssumeRolePolicyDocument(adminPrincipals).Statement[0],
			policies.AssumeRolePolicyDocument(&policies.Principal{
				AWS: []string{users.Arn(
					aws.StringValue(account.Id),
					users.CredentialFactory,
				)},
				Service: []string{"ec2.amazonaws.com"},
			}).Statement[0],
			policies.AssumeRolePolicyDocument(&policies.Principal{
				Federated: []string{saml.Arn},
			}).Statement[0],
		},
	}
	//log.Printf("%+v", assumeRolePolicyDocument)
	if _, err := admin.EnsureAdministratorRole(svc, assumeRolePolicyDocument); err != nil {
		log.Fatal(err)
	}
	if _, err := admin.EnsureAuditorRole(svc, assumeRolePolicyDocument); err != nil {
		log.Fatal(err)
	}
	ui.Stop("ok")

	// Google asks GSuite admins to set custom attributes user by user.  Help
	// these poor souls out by at least telling them exactly what value to set.
	if name == Google {
		ui.Printf(
			`set the AWS/Role custom attribute in GSuite for every authorized AWS Console user to "%s,%s"`,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
			saml.Arn,
		)
		if _, err := ui.Prompt("press <enter> after you've configured at least one GSuite user (so you can test this)"); err != nil {
			log.Fatal(err)
		}
	}

	// Give Okta a user that can enumerate the roles it can assume.  Only Okta
	// needs this.  Google puts more of the burden on GSuite admins.
	if name == Okta {
		ui.Spin("finding or creating a user for Okta to use to enumerate roles")
		user, err := awsiam.EnsureUserWithPolicy(
			svc,
			Okta,
			&policies.Document{
				Statement: []policies.Statement{{
					Action:   []string{"iam:ListAccountAliases", "iam:ListRoles"},
					Resource: []string{"*"},
				}},
			},
		)
		if err != nil {
			log.Fatal(err)
		}
		ui.Stopf("user %s", user.UserName)
		//log.Printf("%+v", user)
		if ok, err := ui.Confirm(
			`answering "yes" will break any existing integration - do you need to configure Okta's AWS integration? (yes/no)`,
		); err != nil {
			log.Fatal(err)
		} else if ok {
			ui.Spin("deleting existing access keys and creating a new one")
			if err := awsiam.DeleteAllAccessKeys(svc, Okta); err != nil {
				log.Fatal(err)
			}
			accessKey, err := awsiam.CreateAccessKey(svc, aws.StringValue(user.UserName))
			if err != nil {
				log.Fatal(err)
			}
			ui.Stop("ok")
			//log.Printf("%+v", accessKey)
			ui.Printf("Okta needs this SAML provider ARN: %s", saml.Arn)
			ui.Printf(".. and this access key ID: %s", accessKey.AccessKeyId)
			ui.Printf("...and this secret access key: %s", accessKey.SecretAccessKey)
			if _, err := ui.Prompt("press <enter> after you've updated your Okta configuration"); err != nil {
				log.Fatal(err)
			}
		}
	}

	return name
}
