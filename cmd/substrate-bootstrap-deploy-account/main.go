package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func main() {
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	cmdutil.MustChdir()
	flag.Parse()
	version.Flag()

	sess, err := awssessions.InSpecialAccount(accounts.Deploy, roles.DeployAdministrator, awssessions.Config{
		FallbackToRootCredentials: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	accountId := aws.StringValue(awssts.MustGetCallerIdentity(sts.New(sess)).Account)
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if err != nil {
		log.Fatal(err)
	}
	prefix := choices.Prefix()

	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect every AWS region in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each region")
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, accounts.Deploy, region)

		// TODO setup global and regional modules just like in other accounts

		file := terraform.NewFile()
		name := fmt.Sprintf("%s-deploy-artifacts-%s", prefix, region) // S3 bucket names are still global
		policy := &policies.Document{
			Statement: []policies.Statement{
				policies.Statement{
					Principal: &policies.Principal{AWS: []string{accountId}},
					Action:    []string{"s3:*"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", name),
						fmt.Sprintf("arn:aws:s3:::%s/*", name),
					},
				},
				policies.Statement{
					Principal: &policies.Principal{AWS: []string{"*"}},
					Action:    []string{"s3:GetObject", "s3:ListBucket", "s3:PutObject"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", name),
						fmt.Sprintf("arn:aws:s3:::%s/*", name),
					},
					Condition: policies.Condition{"StringEquals": {"aws:PrincipalOrgID": aws.StringValue(org.Id)}},
				},
			},
		}
		tags := terraform.Tags{
			Name:   name,
			Region: region,
		}
		file.Push(terraform.S3Bucket{
			Bucket: terraform.Q(tags.Name),
			Label:  terraform.Label(tags),
			Policy: terraform.Q(policy.MustMarshal()),
			Tags:   tags,
		})
		if err := file.Write(filepath.Join(dirname, "main.tf")); err != nil {
			log.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Push(terraform.ProviderFor(
			region,
			roles.Arn(accountId, roles.DeployAdministrator),
		))
		providersFile.Push(terraform.GlobalProvider(
			roles.Arn(accountId, roles.DeployAdministrator),
		))
		networkAccount, err := awsorgs.FindSpecialAccount(organizations.New(awssessions.Must(awssessions.InManagementAccount(
			roles.OrganizationReader,
			awssessions.Config{},
		))), accounts.Network)
		if err != nil {
			log.Fatal(err)
		}
		providersFile.Push(terraform.NetworkProviderFor(
			region,
			roles.Arn(aws.StringValue(networkAccount.Id), roles.Auditor),
		))
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Root(dirname, region); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Print("next, commit root-modules/deploy/ to version control, then run substrate-create-admin-account")
}
