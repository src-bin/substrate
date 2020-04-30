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
			"SubstrateOrganizationAdministrator",
			`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`,
		)
		if err != nil {
			log.Fatal(err)
		}
		//log.Printf("%+v", user)

		if err := awsiam.DeleteAllAccessKeys(
			svc,
			"SubstrateOrganizationAdministrator",
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
			"SubstrateOrganizationAdministrator",
		) // TODO ensure this succeeds even when we exit via log.Fatal

		sess = awsutil.NewSessionExplicit(
			aws.StringValue(accessKey.AccessKeyId),
			aws.StringValue(accessKey.SecretAccessKey),
			region,
		)

		// Inconceivably, the new access key probably isn't usable for a
		// little while so we have to sit and spin before using it.
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
		`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`,
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
	if err := awsorgs.EnsurePolicy(
		svc,
		root,
		"SubstrateTaggingPolicy",
		awsorgs.TAG_POLICY,
		`{"tags":{}}`,
	); err != nil {
		log.Fatal(err)
	}
	/*
		for policySummary := range awsorgs.ListPolicies(svc, awsorgs.TAG_POLICY) {
			log.Printf("%+v", policySummary)
		}
		//*/

	ui.Stop("ok")

	// Ensure the audit account exists.  This one comes first so we can enable
	// CloudTrail ASAP.
	ui.Spin("finding or creating the audit account")
	account, err := awsorgs.EnsureSpecialAccount(
		svc,
		"audit",
		awsorgs.EmailForAccount(org, "audit"),
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", aws.StringValue(account.Id))
	//log.Printf("%+v", account)

	// Ensure CloudTrail is permanently enabled organization-wide.
	ui.Spin("configuring CloudTrail for your organization (every account, every region)")
	bucketName := fmt.Sprintf("%s-cloudtrail", prefix)
	if err := awss3.EnsureBucket(
		s3.New(sess, &aws.Config{
			Credentials: stscreds.NewCredentials(sess, fmt.Sprintf(
				"arn:aws:iam::%s:role/OrganizationAccountAccessRole",
				aws.StringValue(account.Id),
			)),
			Region: aws.String(region),
		}),
		bucketName,
		region,
		policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Principal: policies.Principal{AWS: []string{aws.StringValue(account.Id)}},
					Action:    []string{"s3:*"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
					},
				},
				policies.Statement{
					Principal: policies.Principal{Service: []string{"cloudtrail.amazonaws.com"}},
					Action:    []string{"s3:GetBucketAcl", "s3:PutObject"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/AWSLogs/*", bucketName),
					},
				},
				policies.Statement{
					Principal: policies.Principal{AWS: []string{"*"}},
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
