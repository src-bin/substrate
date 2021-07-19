package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/ram"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awscloudtrail"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsram"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

const (
	ServiceControlPolicyName = "SubstrateServiceControlPolicy"
	TagPolicyName            = "SubstrateTaggingPolicy"
	TrailName                = "GlobalMultiRegionOrganizationTrail"
)

func main() {
	cmdutil.Chdir()
	flag.Parse()
	version.Flag()

	prefix := choices.Prefix()

	region := choices.DefaultRegion()

	sess, err := awssessions.InManagementAccount(roles.OrganizationAdministrator, awssessions.Config{
		BootstrappingManagementAccount: true,
		FallbackToRootCredentials:      true,
		Region:                         region,
	})
	if err != nil {
		log.Fatal(err)
	}

	svc := organizations.New(sess)

	// Ensure this account is (in) an organization.
	ui.Spin("finding or creating your organization")
	org, err := awsorgs.DescribeOrganization(svc)
	if awsutil.ErrorCodeIs(err, awsorgs.AlreadyInOrganizationException) {
		err = nil // we presume this is the management account, to be proven later
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
	if err := accounts.WriteManagementAccountIdToDisk(aws.StringValue(org.MasterAccountId)); err != nil {
		log.Fatal(err)
	}
	ui.Stopf("organization %s", org.Id)
	//log.Printf("%+v", org)

	// TODO EnableAllFeatures, which is complicated but necessary in case an
	// organization was created as merely a consolidated billing organization.

	// Ensure this is, indeed, the organization's management account.  This is
	// almost certainly redundant but I can't be bothered to read the reams
	// of documentation that it would take to prove this beyond a shadow of a
	// doubt so here we are wearing a belt and suspenders.
	ui.Spin("confirming the access key is from the organization's management account")
	callerIdentity := awssts.MustGetCallerIdentity(sts.New(sess))
	org, err = awsorgs.DescribeOrganization(svc)
	if err != nil {
		log.Fatal(err)
	}
	if aws.StringValue(callerIdentity.Account) != aws.StringValue(org.MasterAccountId) {
		log.Fatalf(
			"access key is from account %v instead of your organization's management account, %v",
			aws.StringValue(callerIdentity.Account),
			aws.StringValue(org.MasterAccountId),
		)
	}
	ui.Stop("ok")
	//log.Printf("%+v", callerIdentity)
	//log.Printf("%+v", org)

	// Ensure the audit account exists.  This one comes first so we can enable
	// CloudTrail ASAP.  We might be _too_ fast, though, so we accommodate AWS
	// being a little slow in bootstrapping the organization for this the first
	// of several account creations.
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
				{
					Principal: &policies.Principal{AWS: []string{aws.StringValue(auditAccount.Id)}},
					Action:    []string{"s3:*"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
					},
				},
				{
					Principal: &policies.Principal{Service: []string{"cloudtrail.amazonaws.com"}},
					Action:    []string{"s3:GetBucketAcl", "s3:PutObject"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", bucketName),
						fmt.Sprintf("arn:aws:s3:::%s/AWSLogs/*", bucketName),
					},
				},
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

	// Tag the management account.
	ui.Spin("tagging the management account")
	if err := awsorgs.Tag(svc, aws.StringValue(org.MasterAccountId), map[string]string{
		tags.Manager:                 tags.Substrate,
		tags.SubstrateSpecialAccount: accounts.Management,
		tags.SubstrateVersion:        version.Version,
	}); err != nil {
		log.Fatal(err)
	}
	ui.Stop("ok")

	// Render a "cheat sheet" of sorts that has all the account numbers, role
	// names, and role ARNs that folks might need to get the job done.
	if err := accounts.CheatSheet(svc); err != nil {
		log.Fatal(err)
	}

	ui.Spin("configuring your organization's service control and tagging policies")

	// The management account isn't the organization, though.  It's just an account.
	// To affect the entire organization, we need its root.
	root, err := awsorgs.Root(svc)
	if err != nil {
		log.Fatal(err)
	}

	// Ensure service control policies are enabled and that Substrate's is
	// attached and up-to-date.
	if err := awsorgs.EnablePolicyType(svc, awsorgs.SERVICE_CONTROL_POLICY); err != nil {
		log.Fatal(err)
	}
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

	admin.EnsureAdminRolesAndPolicies(sess)

	ui.Print("next, commit substrate.* to version control, then run substrate-bootstrap-network-account")

	// At the very, very end, when we're exceedingly confident in the
	// capabilities of the other accounts, detach the FullAWSAccess policy
	// from the management account.
	//
	// It's not clear to me that this is EVER a state we'll reach.  It's very
	// tough to give away one's ultimate get-out-of-jail-free card, after all.
	//
	// A safer step would be to attach a policy that allowed re-attaching the
	// FullAWSAccess policy before detaching it.  That would prevent accidental
	// use of the management account without being a "one-way door."

}
