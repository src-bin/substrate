package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	Administrator = "Administrator"
	Auditor       = "Auditor"
	Okta          = "Okta"
)

func okta(sess *session.Session, account *organizations.Account, metadata string) {
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if err != nil {
		log.Fatal(err)
	}

	svc := iam.New(sess, &aws.Config{
		Credentials: stscreds.NewCredentials(sess, roles.Arn(
			aws.StringValue(account.Id),
			roles.OrganizationAccountAccessRole,
		)),
	})

	ui.Spin("configuring Okta as your organization's identity provider")
	saml, err := awsiam.EnsureSAMLProvider(svc, Okta, metadata)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("provider %s", saml.Arn)
	//log.Printf("%+v", saml)

	// Give Okta some entrypoints in the Admin account.
	ui.Spin("finding or creating roles for Okta to use in the ops account")
	assumeRolePolicyDocument := &policies.Document{
		Statement: []policies.Statement{
			policies.AssumeRolePolicyDocument(&policies.Principal{
				AWS: []string{
					aws.StringValue(org.MasterAccountId),
					roles.Arn(
						aws.StringValue(account.Id),
						"substrate-credential-factory",
					),
					users.Arn(
						aws.StringValue(org.MasterAccountId),
						users.OrganizationAdministrator,
					),
				},
			}).Statement[0],
			policies.AssumeRolePolicyDocument(&policies.Principal{
				Federated: []string{saml.Arn},
			}).Statement[0],
		},
	}
	_, err = awsiam.EnsureRoleWithPolicy(
		svc,
		Administrator,
		assumeRolePolicyDocument,
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
		log.Fatal(err)
	}
	_, err = awsiam.EnsureRoleWithPolicy(
		svc,
		Auditor, // TODO allow it to assume roles but set a permission boundary to keep it read-only
		assumeRolePolicyDocument,
		&policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Action: []string{
						"cloudformation:GetTemplate",
						"dynamodb:BatchGetItem",
						"dynamodb:GetItem",
						"dynamodb:Query",
						"dynamodb:Scan",
						"ec2:GetConsoleOutput",
						"ec2:GetConsoleScreenshot",
						"ecr:BatchGetImage",
						"ecr:GetAuthorizationToken",
						"ecr:GetDownloadUrlForLayer",
						"kinesis:Get*",
						"lambda:GetFunction",
						"logs:GetLogEvents",
						"s3:GetObject",
						"sdb:Select*",
						"sqs:ReceiveMessage",
					},
					Effect:   policies.Deny, // <https://alestic.com/2015/10/aws-iam-readonly-too-permissive/>
					Resource: []string{"*"},
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := awsiam.AttachRolePolicy(
		svc,
		Auditor,
		"arn:aws:iam::aws:policy/ReadOnlyAccess",
	); err != nil {
		log.Fatal(err)
	}
	ui.Stop("ok")

	// And give Okta a user that can enumerate the roles it can assume.
	ui.Spin("finding or creating a user for Okta to use to enumerate roles")
	user, err := awsiam.EnsureUserWithPolicy(
		svc,
		Okta,
		&policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Action:   []string{"iam:ListAccountAliases", "iam:ListRoles"},
					Resource: []string{"*"},
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("user %s", user.UserName)
	//log.Printf("%+v", user)
	if ok, err := ui.Confirm("do you need to configure Okta's AWS integration? (yes or no)"); err != nil {
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
		_, err = ui.Prompt("press ENTER after you've updated your Okta configuration")
		if err != nil {
			log.Fatal(err)
		}
	}

}
