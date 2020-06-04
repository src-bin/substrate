package main

import (
	"fmt"
	"log"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/s3config"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
)

const TerraformDirname = "deploy-account"

func main() {

	sess, err := awssessions.InSpecialAccount(
		accounts.Deploy,
		roles.DeployAdministrator,
		awssessions.Config{},
	)
	if err != nil {
		ui.Print("unable to assume the DeployAdministrator role, which means this is probably your first time bootstrapping your deploy account; please provide an access key from your master AWS account")
		accessKeyId, secretAccessKey := awsutil.ReadAccessKeyFromStdin()
		ui.Printf("using access key %s", accessKeyId)
		sess, err = awssessions.InSpecialAccount(
			accounts.Deploy,
			roles.DeployAdministrator,
			awssessions.Config{
				AccessKeyId:     accessKeyId,
				SecretAccessKey: secretAccessKey,
			},
		)
	}
	if err != nil {
		log.Fatal(err)
	}

	prefix := s3config.Prefix()

	callerIdentity := awssts.MustGetCallerIdentity(sts.New(sess))
	accountId := aws.StringValue(callerIdentity.Account)

	// Write (or rewrite) some Terraform providers to make everything usable.
	providers := terraform.Provider{
		AccountId:   accountId,
		RoleName:    roles.DeployAdministrator,
		SessionName: "Terraform",
	}.AllRegions()
	if err := providers.Write(path.Join(TerraformDirname, "providers.tf")); err != nil {
		log.Fatal(err)
	}

	// Write (or rewrite) Terraform resources that creates the S3 bucket we
	// will use to shuttle artifacts between Environments and Qualities.
	blocks := terraform.NewBlocks()
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
				/*
					policies.Statement{
						Principal: &policies.Principal{AWS: []string{"*"}},
						Action:    []string{"s3:GetObject", "s3:ListBucket", "s3:PutObject"},
						Resource: []string{
							fmt.Sprintf("arn:aws:s3:::%s", name),
							fmt.Sprintf("arn:aws:s3:::%s/*", name),
						},
						Condition: policies.Condition{"StringEquals": {"aws:PrincipalOrgID": aws.StringValue(org.Id)}},
					},
				*/
			},
		}
		tags := terraform.Tags{
			Name:   name,
			Region: region,
		}
		blocks.Push(terraform.S3Bucket{
			Bucket:   terraform.Q(tags.Name),
			Label:    terraform.Label(tags),
			Policy:   terraform.Q(policy.MustMarshal()),
			Provider: terraform.ProviderAliasFor(region),
			Tags:     tags,
		})
	}
	if err := blocks.Write(path.Join(TerraformDirname, "s3.tf")); err != nil {
		log.Fatal(err)
	}

	// Format all the Terraform code you can possibly find.
	if err := terraform.Fmt(); err != nil {
		log.Fatal(err)
	}

	// Generate a Makefile in the root Terraform module then apply the generated
	// Terraform code.
	if err := terraform.Makefile(TerraformDirname); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Init(TerraformDirname); err != nil {
		log.Fatal(err)
	}
	if err := terraform.Apply(TerraformDirname); err != nil {
		log.Fatal(err)
	}

}
