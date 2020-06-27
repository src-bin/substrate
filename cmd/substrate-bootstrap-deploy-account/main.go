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
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
)

const TerraformDirname = "deploy-account"

func main() {
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	flag.Parse()

	sess, err := awssessions.InSpecialAccount(accounts.Deploy, roles.DeployAdministrator, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}

	prefix := choices.Prefix()

	callerIdentity := awssts.MustGetCallerIdentity(sts.New(sess))
	accountId := aws.StringValue(callerIdentity.Account)
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if err != nil {
		log.Fatal(err)
	}

	// Write (or rewrite) some Terraform providers to make everything usable.
	providersFile := terraform.NewFile()
	providersFile.PushAll(terraform.Provider{
		AccountId:   accountId,
		RoleName:    roles.DeployAdministrator,
		SessionName: "Terraform",
	}.AllRegionsAndGlobal())
	networkAccount, err := awsorgs.FindSpecialAccount(organizations.New(awssessions.Must(awssessions.InMasterAccount(
		roles.OrganizationReader,
		awssessions.Config{},
	))), accounts.Network)
	if err != nil {
		log.Fatal(err)
	}
	providersFile.PushAll(terraform.Provider{
		AccountId:   aws.StringValue(networkAccount.Id),
		AliasSuffix: "network",
		RoleName:    roles.Auditor,
		SessionName: "Terraform",
	}.AllRegions())
	if err := providersFile.Write(filepath.Join(TerraformDirname, "providers.tf")); err != nil {
		log.Fatal(err)
	}

	// Write (or rewrite) Terraform resources that creates the S3 bucket we
	// will use to shuttle artifacts between environments and qualities.
	file := terraform.NewFile()
	for _, region := range regions.Selected() {
		name := fmt.Sprintf("%s-deploy-artifacts-%s", prefix, region) // including region because S3 bucket names are still global
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
			Bucket:   terraform.Q(tags.Name),
			Label:    terraform.Label(tags),
			Policy:   terraform.Q(policy.MustMarshal()),
			Provider: terraform.ProviderAliasFor(region),
			Tags:     tags,
		})
	}
	if err := file.Write(filepath.Join(TerraformDirname, "s3.tf")); err != nil {
		log.Fatal(err)
	}

	// TODO setup global and regional modules just like in other accounts

	// Format all the Terraform code you can possibly find.
	if err := terraform.Fmt(); err != nil {
		log.Fatal(err)
	}

	// Generate a Makefile in the root Terraform module then apply the generated
	// Terraform code.
	if err := terraform.Root(TerraformDirname, sess); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Init(TerraformDirname); err != nil {
		log.Fatal(err)
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	} else {
		if err := terraform.Apply(TerraformDirname); err != nil {
			log.Fatal(err)
		}
	}

	ui.Print("next, commit deploy-account/ to version control, then run substrate-create-admin-account")
}
