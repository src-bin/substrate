package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/ram"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscloudtrail"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsram"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/s3config"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	CloudTrailRegionFilename = "substrate.CloudTrail-region"
	ServiceControlPolicyName = "SubstrateServiceControlPolicy"
	TagPolicyName            = "SubstrateTaggingPolicy"
	TrailName                = "GlobalMultiRegionOrganizationTrail"
)

func main() {

	ui.Print("time to bootstrap the AWS organization so we need an access key from your new master AWS account")
	accessKeyId, secretAccessKey := awsutil.ReadAccessKeyFromStdin()
	ui.Printf("using access key %s", accessKeyId)

	prefix := s3config.Prefix()

	region, err := ui.PromptFile(
		CloudTrailRegionFilename,
		"what region should host the S3 bucket that stores your CloudTrail logs?",
	)
	if err != nil {
		log.Fatal(err)
	}
	if !regions.IsRegion(region) {
		log.Fatalf("%s is not an AWS region", region)
	}
	ui.Printf("using region %s", region)

	sess, err := awssessions.InMasterAccount(roles.OrganizationAdministrator, awssessions.Config{
		AccessKeyId:     accessKeyId,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	})
	if err != nil {
		log.Fatal(err)
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
	ui.Stopf("organization %s", org.Id)
	//log.Printf("%+v", org)

	// TODO EnableAllFeatures, which is complicated but necessary in case an
	// organization was created as merely a consolidated billing organization.

	// Ensure this is, indeed, the organization's master account.  This is
	// almost certainly redundant but I can't be bothered to read the reams
	// of documentation that it would take to prove this beyond a shadow of a
	// doubt so here we are wearing a belt and suspenders.
	ui.Spin("confirming the access key is from the organization's master account")
	callerIdentity := awssts.MustGetCallerIdentity(sts.New(sess))
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

	// Ensure the audit account exists.  This one comes first so we can enable
	// CloudTrail ASAP.
	ui.Spin("finding or creating the audit account")
	auditAccount, err := awsorgs.EnsureSpecialAccount(svc, accounts.Audit)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", auditAccount.Id)
	//log.Printf("%+v", auditAccount)

	// Ensure CloudTrail is permanently enabled organization-wide.
	ui.Spin("configuring CloudTrail for your organization (every account, every region)")
	bucketName := fmt.Sprintf("%s-cloudtrail", prefix)
	if err := awss3.EnsureBucket(
		s3.New(sess, &aws.Config{
			Credentials: stscreds.NewCredentials(sess, roles.Arn(
				aws.StringValue(auditAccount.Id),
				roles.OrganizationAccountAccessRole,
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
				/*
					policies.Statement{
						Principal: &policies.Principal{AWS: []string{"*"}},
						Action:    []string{"s3:GetObject", "s3:ListBucket"},
						Resource: []string{
							fmt.Sprintf("arn:aws:s3:::%s", bucketName),
							fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
						},
						Condition: policies.Condition{"StringEquals": {"aws:PrincipalOrgID": aws.StringValue(org.Id)}},
					},
				*/
			},
		},
	); err != nil {
		log.Fatal(err)
	}
	if err := awsorgs.EnableAWSServiceAccess(svc, "cloudtrail.amazonaws.com"); err != nil {
		log.Fatal(err)
	}
	trail, err := awscloudtrail.EnsureTrail(cloudtrail.New(sess), TrailName, bucketName)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("bucket %s, trail %s", bucketName, trail.Name)

	// Ensure the deploy and network accounts exist.
	ui.Spinf("finding or creating the deploy account")
	deployAccount, err := awsorgs.EnsureSpecialAccount(svc, accounts.Deploy)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", deployAccount.Id)
	//log.Printf("%+v", account)
	ui.Spinf("finding or creating the network account")
	networkAccount, err := awsorgs.EnsureSpecialAccount(svc, accounts.Network)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", networkAccount.Id)
	//log.Printf("%+v", networkAccount)

	// Render a "cheat sheet" of sorts that has all the account numbers, role
	// names, and role ARNs that folks might need to get the job done.
	if err := accounts.CheatSheet(
		org,
		auditAccount,
		deployAccount,
		networkAccount,
	); err != nil {
		log.Fatal(err)
	}

	// Tag the master account.
	ui.Spin("tagging the master account")
	if err := awsorgs.Tag(svc, aws.StringValue(org.MasterAccountId), map[string]string{
		tags.Manager:                 tags.Substrate,
		tags.SubstrateSpecialAccount: accounts.Master,
		tags.SubstrateVersion:        version.Version,
	}); err != nil {
		log.Fatal(err)
	}
	ui.Stop("ok")

	ui.Spin("configuring your organization's service control and tagging policies")

	// The master account isn't the organization, though.  It's just an account.
	// To affect the entire organization, we need its root.
	root, err := awsorgs.Root(svc)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", root)

	// Ensure service control policies are enabled and that Substrate's is
	// attached and up-to-date.
	if err := awsorgs.EnsurePolicy(
		svc,
		root,
		ServiceControlPolicyName,
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
			TagPolicyName,
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

	// Enable resource sharing throughout the organization.
	ui.Spin("enabling resource sharing throughout your organization")
	if err := awsram.EnableSharingWithAwsOrganization(ram.New(sess)); err != nil {
		log.Fatal(err)
	}
	ui.Stop("ok")

	// Gather lists of accounts.  These are used below in configuring policies
	// to allow cross-account access.  On the first run they're basically
	// no-ops but on subsequent runs this is key to not undoing the work of
	// substrate-create-account and substrate-create-admin-account.
	adminAccounts, err := awsorgs.FindAccountsByDomain(svc, accounts.Admin)
	if err != nil {
		log.Fatal(err)
	}
	adminPrincipals := make([]string, len(adminAccounts)+2) // +2 for the master account and its IAM user
	for i, account := range adminAccounts {
		adminPrincipals[i] = roles.Arn(aws.StringValue(account.Id), roles.Administrator)
	}
	adminPrincipals[len(adminPrincipals)-2] = aws.StringValue(org.MasterAccountId)
	adminPrincipals[len(adminPrincipals)-1] = fmt.Sprintf("arn:aws:iam::%s:user/%s", aws.StringValue(org.MasterAccountId), roles.OrganizationAdministrator)
	sort.Strings(adminPrincipals) // to avoid spurious policy diffs
	log.Printf("%+v", adminPrincipals)
	allAccounts, err := awsorgs.ListAccounts(svc)
	if err != nil {
		log.Fatal(err)
	}
	allAccountIds := make([]string, len(allAccounts))
	for i, account := range allAccounts {
		allAccountIds[i] = aws.StringValue(account.Id)
	}
	sort.Strings(allAccountIds) // to avoid spurious policy diffs
	log.Printf("%+v", allAccountIds)

	// Admin accounts, once they exist, are going to need to be able to assume
	// a role in the master account.  Because no admin accounts exist yet, this
	// is something of a no-op but it provides a place for them to attach
	// themselves as they're created.
	ui.Spin("finding or creating a role to allow admin accounts to administer your organization")
	role, err := awsiam.EnsureRoleWithPolicy(
		iam.New(sess),
		roles.OrganizationAdministrator,
		&policies.Principal{AWS: adminPrincipals},
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
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	// Ensure admin accounts can find other accounts in the organization without full administrative
	// privileges.  This should be directly possible via organizations:RegisterDelegatedAdministrator
	// but that API appears to just not work the way
	// <https://docs.aws.amazon.com/organizations/latest/userguide/orgs_integrated-services-list.html>
	// implies it does.  As above, this is almost a no-op for now because, while it grants access to
	// the special accounts, no admin accounts exist yet.
	ui.Spin("finding or creating a role to allow account discovery within your organization")
	role, err = awsiam.EnsureRoleWithPolicy(
		iam.New(sess),
		roles.OrganizationReader,
		&policies.Principal{AWS: allAccountIds},
		&policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Action:   []string{"organizations:ListAccounts"},
					Resource: []string{"*"},
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	// Ensure the master account can get into the deploy and network accounts
	// the same way admin accounts will in the future.  This will make deploy
	// and network account boostrapping (the next steps) easier.
	ui.Spin("finding or creating a role to allow admin accounts to administer your deploy artifacts")
	role, err = awsiam.EnsureRoleWithPolicy(
		iam.New(awssessions.AssumeRole(
			sess,
			aws.StringValue(deployAccount.Id),
			roles.OrganizationAccountAccessRole,
		)),
		roles.DeployAdministrator,
		&policies.Principal{AWS: adminPrincipals},
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
	ui.Stopf("role %s", role.Name)
	ui.Spin("finding or creating a role to allow admin accounts to administer your networks")
	//log.Printf("%+v", role)
	role, err = awsiam.EnsureRoleWithPolicy(
		iam.New(awssessions.AssumeRole(
			sess,
			aws.StringValue(networkAccount.Id),
			roles.OrganizationAccountAccessRole,
		)),
		roles.NetworkAdministrator,
		&policies.Principal{AWS: adminPrincipals},
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
	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)

	ui.Print("next, run substrate-bootstrap-deploy-account and substrate-bootstrap-network-account")

	//ui.Print("until we get you an EC2 instance profile, here's your way into the ops account (good for one hour)")
	//awssts.Export(awssts.AssumeRole(sts.New(sess), roles.Arn(aws.StringValue(opsAccount.Id), roles.OrganizationAccountAccessRole)))

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
