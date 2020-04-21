package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/version"
)

func main() {
	log.SetFlags(log.Lshortfile)

	fmt.Println("time to bootstrap the AWS organization so we need an access key from your new AWS master account")
	accessKeyId, secretAccessKey := awsutil.ReadAccessKeyFromStdin()
	fmt.Printf("proceeding with access key ID %s\n", accessKeyId)

	sess := awsutil.NewSessionExplicit(accessKeyId, secretAccessKey)

	svc := organizations.New(sess)

	// Ensure this account is (in) an organization.
	org, err := awsorgs.DescribeOrganization(svc)
	if awsutil.ErrorCodeIs(err, awsorgs.AlreadyInOrganizationException) {
		// Here we presume this is the master account, to be proven later.
	} else if awsutil.ErrorCodeIs(err, awsorgs.AWSOrganizationsNotInUseException) {

		// Create the organization since it doesn't yet exist.
		org, err = awsorgs.CreateOrganization(svc)
		if err != nil {
			log.Fatal(err)
		}

	} else if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", org)

	// Ensure this is, indeed, the organization's master account.  This is
	// almost certainly redundant but I can't be bothered to read the reams
	// of documentation that it would take to prove this beyond a shadow of a
	// doubt so here we are wearing a belt and suspenders.
	callerIdentity := awssts.GetCallerIdentity(sts.New(sess))
	//log.Printf("%+v", callerIdentity)
	org, err = awsorgs.DescribeOrganization(svc)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", org)
	if aws.StringValue(callerIdentity.Account) != aws.StringValue(org.MasterAccountId) {
		log.Fatalf(
			"access key is from account %v instead of the organization's master account, %v",
			aws.StringValue(callerIdentity.Account),
			aws.StringValue(org.MasterAccountId),
		)
	}

	// Tag the master account.
	if err := awsorgs.Tag(svc, aws.StringValue(org.MasterAccountId), map[string]string{
		"Manager":                 "Substrate",
		"SubstrateSpecialAccount": "master",
		"SubstrateVersion":        version.Version,
	}); err != nil {
		log.Fatal(err)
	}

	// The master account isn't the organization, though.  It's just an account.
	// To affect the entire organization, we need its root.
	root := awsorgs.Root(svc)
	//log.Printf("%+v", root)

	// Ensure service control policies are enabled and that Substrate's is
	// attached and up-to-date.
	if err := awsorgs.EnsurePolicy(
		svc,
		root,
		"Substrate",
		awsorgs.SERVICE_CONTROL_POLICY,
		`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`,
	); err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", awsorgs.ListPolicies(svc, awsorgs.SERVICE_CONTROL_POLICY))

	// Ensure tagging policies are enabled and that Substrate's is attached
	// and up-to-date.
	if err := awsorgs.EnsurePolicy(
		svc,
		root,
		"Substrate",
		awsorgs.TAG_POLICY,
		`{"tags":{}}`,
	); err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", awsorgs.ListPolicies(svc, awsorgs.TAG_POLICY))

	// Ensure the audit, deploy, network, and ops accounts exist.
	for _, name := range []string{"audit", "deploy", "network", "ops"} {
		account, err := awsorgs.EnsureSpecialAccount(
			svc,
			name,
			strings.Replace(
				aws.StringValue(org.MasterAccountEmail),
				"@",
				fmt.Sprintf("+%s@", name),
				1,
			),
		)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%+v", account)
	}

	// At the very, very end, when we're exceedingly confident in the
	// capabilities of the other accounts, detach the FullAWSAccess policy
	// from the master account.
	//
	// It's not clear to me that this is EVER a state we'll reach.  It's very
	// tough to give away one's ultimate get-out-of-jail-free card, after all.

}
