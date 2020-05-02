package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awscloudtrail"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func main() {
	log.SetFlags(log.Lshortfile)

	ui.Print("time to bootstrap the AWS organization so we need an access key from your new master AWS account")
	accessKeyId, secretAccessKey := awsutil.ReadAccessKeyFromStdin()
	ui.Printf("using access key %s", accessKeyId)

	prefix, err := ui.PromptFile(
		"substrate.prefix",
		"what prefix do you want to use for global names like S3 buckets?",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using prefix %s", prefix)

	region, err := ui.PromptFile(
		"substrate.region",
		"what region should host your dev/ops EC2 instances, CloudTrail logs, etc?",
	)
	if err != nil {
		log.Fatal(err)
	}
	if !awsutil.IsRegion(region) {
		log.Fatalf("%s is not an AWS region", region)
	}
	ui.Printf("using region %s", region)

	// TODO need to launch an editor for this because a line reader isn't going to cut it
	metadata, err := ui.PromptFile(
		"substrate.okta.xml",
		"paste your identity provider metadata XML from Okta\n",
	)
	if err != nil {
		log.Fatal(err)
	}
	// TODO do some light validation of the XML

	sess := awsutil.NewSessionExplicit(accessKeyId, secretAccessKey, region)

	// Switch to an IAM user so that we can assume roles in other accounts.
	callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		log.Fatal(err)
	}
	if strings.HasSuffix(aws.StringValue(callerIdentity.Arn), ":root") {
		ui.Spin("switching to an IAM user that can assume roles in other accounts")
		svc := iam.New(sess)

		user, err := awsiam.EnsureUserWithPolicy(
			svc,
			"OrganizationAdministrator",
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
		//log.Printf("%+v", user)

		if err := awsiam.DeleteAllAccessKeys(
			svc,
			"OrganizationAdministrator",
		); err != nil {
			log.Fatal(err)
		}

		accessKey, err := awsiam.CreateAccessKey(svc, aws.StringValue(user.UserName))
		if err != nil {
			log.Fatal(err)
		}
		//log.Printf("%+v", accessKey)
		defer awsiam.DeleteAllAccessKeys(
			svc,
			"OrganizationAdministrator",
		) // TODO ensure this succeeds even when we exit via log.Fatal

		sess = awsutil.NewSessionExplicit(
			aws.StringValue(accessKey.AccessKeyId),
			aws.StringValue(accessKey.SecretAccessKey),
			region,
		)

		// Inconceivably, the new access key probably isn't usable for a
		// little while so we have to sit and spin before using it.
		//
		// TODO and even with this loop we can sometimes jump the gun.
		for {
			_, err := awssts.GetCallerIdentity(sts.New(sess))
			if err == nil {
				break
			}
			if !awsutil.ErrorCodeIs(err, awssts.InvalidClientTokenId) {
				log.Fatal(err)
			}
			time.Sleep(1e9) // TODO exponential backoff
		}

		ui.Stopf("switched to access key %s", aws.StringValue(accessKey.AccessKeyId))
	}

	svc := organizations.New(sess)

	// Ensure this account is (in) an organization.
	ui.Spin("finding or creating your organization")
	org, err := awsorgs.DescribeOrganization(svc)
	if awsutil.ErrorCodeIs(err, awsorgs.AlreadyInOrganizationException) {
		err = nil // we presume this is the master account, to be proven later
	}
	if awsutil.ErrorCodeIs(err, awsorgs.AWSOrganizationsNotInUseException) {

		// Create the organization since it doesn't yet exist.
		org, err = awsorgs.CreateOrganization(svc)
		if err != nil {
			log.Fatal(err)
		}

	}
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("organization %s", aws.StringValue(org.Id))
	//log.Printf("%+v", org)

	// TODO EnableAllFeatures, which is complicated but necessary in case an
	// organization was created as merely a consolidated billing organization.

	// Ensure this is, indeed, the organization's master account.  This is
	// almost certainly redundant but I can't be bothered to read the reams
	// of documentation that it would take to prove this beyond a shadow of a
	// doubt so here we are wearing a belt and suspenders.
	ui.Spin("confirming the access key is from the organization's master account")
	callerIdentity, err = awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		log.Fatal(err)
	}
	org, err = awsorgs.DescribeOrganization(svc)
	if err != nil {
		log.Fatal(err)
	}
	if aws.StringValue(callerIdentity.Account) != aws.StringValue(org.MasterAccountId) {
		log.Fatalf(
			"access key is from account %v instead of your organization's master account, %v",
			aws.StringValue(callerIdentity.Account),
			aws.StringValue(org.MasterAccountId),
		)
	}
	ui.Stop("ok")
	//log.Printf("%+v", callerIdentity)
	//log.Printf("%+v", org)

	// Tag the master account.
	ui.Spin("tagging the master account")
	if err := awsorgs.Tag(svc, aws.StringValue(org.MasterAccountId), map[string]string{
		"Manager":                 "Substrate",
		"SubstrateSpecialAccount": "master",
		"SubstrateVersion":        version.Version,
	}); err != nil {
		log.Fatal(err)
	}
	ui.Stop("ok")

	ui.Spin("configuring your organization's service control and tagging policies")

	// The master account isn't the organization, though.  It's just an account.
	// To affect the entire organization, we need its root.
	root := awsorgs.Root(svc)
	//log.Printf("%+v", root)

	// Ensure service control policies are enabled and that Substrate's is
	// attached and up-to-date.
	if err := awsorgs.EnsurePolicy(
		svc,
		root,
		"SubstrateServiceControlPolicy",
		awsorgs.SERVICE_CONTROL_POLICY,
		&policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Action:   []string{"*"},
					Resource: []string{"*"},
				},
			},
		},
	); err != nil {
		log.Fatal(err)
	}
	/*
		for policySummary := range awsorgs.ListPolicies(svc, awsorgs.SERVICE_CONTROL_POLICY) {
			log.Printf("%+v", policySummary)
		}
		//*/

	// Ensure tagging policies are enabled and that Substrate's is attached
	// and up-to-date.
	/*
		if err := awsorgs.EnsurePolicy(
			svc,
			root,
			"SubstrateTaggingPolicy",
			awsorgs.TAG_POLICY,
			`{"tags":{}}`,
		); err != nil {
			log.Fatal(err)
		}
	*/
	/*
		for policySummary := range awsorgs.ListPolicies(svc, awsorgs.TAG_POLICY) {
			log.Printf("%+v", policySummary)
		}
		//*/

	ui.Stop("ok")

	// Ensure the audit account exists.  This one comes first so we can enable
	// CloudTrail ASAP.
	ui.Spin("finding or creating the audit account")
	auditAccount, err := awsorgs.EnsureSpecialAccount(
		svc,
		"audit",
		awsorgs.EmailForAccount(org, "audit"),
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", aws.StringValue(auditAccount.Id))
	//log.Printf("%+v", auditAccount)

	// Ensure CloudTrail is permanently enabled organization-wide.
	ui.Spin("configuring CloudTrail for your organization (every account, every region)")
	bucketName := fmt.Sprintf("%s-cloudtrail", prefix)
	if err := awss3.EnsureBucket(
		s3.New(sess, &aws.Config{
			Credentials: stscreds.NewCredentials(sess, fmt.Sprintf(
				"arn:aws:iam::%s:role/OrganizationAccountAccessRole",
				aws.StringValue(auditAccount.Id),
			)),
			Region: aws.String(region),
		}),
		bucketName,
		region,
		&policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Principal: &policies.Principal{AWS: []string{aws.StringValue(auditAccount.Id)}},
					Action:    []string{"s3:*"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
					},
				},
				policies.Statement{
					Principal: &policies.Principal{Service: []string{"cloudtrail.amazonaws.com"}},
					Action:    []string{"s3:GetBucketAcl", "s3:PutObject"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/AWSLogs/*", bucketName),
					},
				},
				policies.Statement{
					Principal: &policies.Principal{AWS: []string{"*"}},
					Action:    []string{"s3:GetObject"},
					Resource:  []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucketName)},
					Condition: policies.Condition{"StringEquals": {"aws:PrincipalOrgID": aws.StringValue(org.Id)}},
				},
			},
		},
	); err != nil {
		log.Fatal(err)
	}
	if err := awsorgs.EnableAWSServiceAccess(svc, "cloudtrail.amazonaws.com"); err != nil {
		log.Fatal(err)
	}
	trail, err := awscloudtrail.EnsureTrail(cloudtrail.New(sess), "GlobalMultiRegionOrganizationTrail", bucketName)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("bucket %s, trail %s", bucketName, aws.StringValue(trail.Name))

	// Ensure the deploy, network, and ops accounts exist.
	for _, name := range []string{"deploy", "network", "ops"} {
		ui.Spinf("finding or creating the %s account", name)
		account, err := awsorgs.EnsureSpecialAccount(
			svc,
			name,
			awsorgs.EmailForAccount(org, name),
		)
		if err != nil {
			log.Fatal(err)
		}
		ui.Stopf("account %s", aws.StringValue(account.Id))
		//log.Printf("%+v", account)
	}

	opsAccount, err := awsorgs.FindSpecialAccount(svc, "ops")
	if err != nil {
		log.Fatal(err)
	}
	config := &aws.Config{
		Credentials: stscreds.NewCredentials(sess, fmt.Sprintf(
			"arn:aws:iam::%s:role/OrganizationAccountAccessRole",
			aws.StringValue(opsAccount.Id),
		)),
		Region: aws.String(region),
	}

	// Ensure the ops account can get back into the master account.
	ui.Spin("finding or creating a role to allow the ops account to administer your organization")
	role, err := awsiam.EnsureRoleWithPolicy(
		iam.New(sess),
		"OrganizationAdministrator",
		&policies.Principal{AWS: []string{aws.StringValue(opsAccount.Id)}},
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
	ui.Stopf("role %s", aws.StringValue(role.RoleName))
	//log.Printf("%+v", role)

	// Configure Okta so we can get into the ops account directly, SSH, etc.
	ui.Spin("configuring Okta as your organization's identity provider")
	saml, err := awsiam.EnsureSAMLProvider(iam.New(sess, config), "Okta", metadata)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("provider %s", saml.Arn)
	//log.Printf("%+v", saml)

	// Give Okta an entrypoint in the ops account.
	ui.Spin("finding or creating a role for Okta to use in the ops account")
	role, err = awsiam.EnsureRoleWithPolicy(
		iam.New(sess, config),
		"Okta",
		&policies.Principal{Federated: []string{saml.Arn}},
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
	ui.Stopf("role %s", aws.StringValue(role.RoleName))
	//log.Printf("%+v", role)

	// And give Okta a user that can enumerate the roles it can assume.
	ui.Spin("finding or creating a user for Okta to use to enumerate roles")
	user, err := awsiam.EnsureUserWithPolicy(
		iam.New(sess, config),
		"Okta",
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
	ui.Stopf("user %s", aws.StringValue(user.UserName))
	//log.Printf("%+v", user)
	if yesno, err := ui.Confirm("do you need to configure Okta's AWS integration? (yes or no)"); err != nil {
		log.Fatal(err)
	} else if yesno == "yes" {
		ui.Spin("deleting existing access keys and creating a new one")
		if err := awsiam.DeleteAllAccessKeys(
			iam.New(sess, config),
			"Okta",
		); err != nil {
			log.Fatal(err)
		}
		accessKey, err := awsiam.CreateAccessKey(iam.New(sess, config), aws.StringValue(user.UserName))
		if err != nil {
			log.Fatal(err)
		}
		ui.Stop("ok")
		//log.Printf("%+v", accessKey)
		ui.Printf("Okta needs this SAML provider ARN: %s", saml.Arn)
		ui.Printf(".. and this access key ID: %s", aws.StringValue(accessKey.AccessKeyId))
		ui.Printf("...and this secret access key: %s", aws.StringValue(accessKey.SecretAccessKey))
		_, err = ui.Prompt("press ENTER after you've updated your Okta configuration")
		if err != nil {
			log.Fatal(err)
		}
	}

	awssts.Export(awssts.AssumeRole(sts.New(sess), fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", aws.StringValue(opsAccount.Id))))

	// At the very, very end, when we're exceedingly confident in the
	// capabilities of the other accounts, detach the FullAWSAccess policy
	// from the master account.
	//
	// It's not clear to me that this is EVER a state we'll reach.  It's very
	// tough to give away one's ultimate get-out-of-jail-free card, after all.
	//
	// A safer step would be to attach a policy that allowed re-attaching the
	// FullAWSAccess policy before detaching it.  That would prevent accidental
	// use of the master account without being a "one-way door."

}