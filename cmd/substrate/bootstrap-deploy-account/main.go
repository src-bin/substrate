package bootstrapdeployaccount

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate bootstrap-deploy-account [-auto-approve|-no-apply]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()

	var err error
	if _, err = cfg.GetCallerIdentity(ctx); err != nil {
		if _, err = cfg.SetRootCredentials(ctx); err != nil {
			ui.Fatal(err)
		}
	}
	cfg = awscfg.Must(cfg.AssumeSpecialRole(
		ctx,
		accounts.Deploy,
		roles.DeployAdministrator,
		time.Hour,
	))
	versionutil.PreventDowngrade(ctx, cfg)

	accountId := aws.ToString(cfg.MustGetCallerIdentity(ctx).Account)
	org, err := cfg.DescribeOrganization(ctx)
	if err != nil {
		ui.Fatal(err)
	}
	prefix := naming.Prefix()

	cfg.Telemetry().FinalAccountId = accountId
	cfg.Telemetry().FinalRoleName = roles.DeployAdministrator

	// Leave the user a place to put their own Terraform code for defining ECR
	// resources or whatever else they want to do in the deploy account.
	if err := terraform.Scaffold(accounts.Deploy, false); err != nil {
		ui.Fatal(err)
	}

	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect every AWS region in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each region")
	}
	{
		dirname := filepath.Join(terraform.RootModulesDirname, accounts.Deploy, regions.Global)
		region := regions.Default()

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Label: terraform.Q(accounts.Deploy),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.UsEast1ProviderAlias: terraform.UsEast1ProviderAlias,
			},
			Source: terraform.Q("../../../modules/", accounts.Deploy, "/global"),
		})
		ui.Must(file.WriteIfNotExists(filepath.Join(dirname, "main.tf")))

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.ARN(accountId, roles.DeployAdministrator),
		))
		providersFile.Add(terraform.UsEast1Provider(
			roles.ARN(accountId, roles.DeployAdministrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, cfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))

		ui.Must(terraform.Init(dirname))

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		ui.Must(err)
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, accounts.Deploy, region)

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Label: terraform.Q(accounts.Deploy),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.NetworkProviderAlias: terraform.NetworkProviderAlias,
			},
			Source: terraform.Q("../../../modules/", accounts.Deploy, "/regional"),
		})
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
					Action: []string{
						"s3:GetBucketLocation",
						"s3:GetObject*",
						"s3:ListBucket*",
					},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", name),
						fmt.Sprintf("arn:aws:s3:::%s/*", name),
					},
					Condition: policies.Condition{"StringEquals": {
						"aws:PrincipalOrgID": []string{aws.ToString(org.Id)},
					}},
				},
				policies.Statement{
					Principal: &policies.Principal{AWS: []string{"*"}},
					Action:    []string{"s3:PutObject*"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s/*", name),
					},
					Condition: policies.Condition{"StringEquals": {
						"aws:PrincipalOrgID": []string{aws.ToString(org.Id)},
						"s3:x-amz-acl":       []string{"bucket-owner-full-control"},
					}},
				},
			},
		}
		tags := terraform.Tags{
			Name:   name,
			Region: region,
		}
		bucket := terraform.S3Bucket{
			Bucket: terraform.Q(tags.Name),
			Label:  terraform.Label(tags),
			Policy: terraform.Q(policy.MustMarshal()),
			Tags:   tags,
		}
		file.Add(bucket)
		file.Add(terraform.S3BucketOwnershipControls{
			Bucket:          terraform.U(bucket.Ref(), ".bucket"),
			Label:           terraform.Label(tags),
			ObjectOwnership: terraform.Q(terraform.BucketOwnerPreferred),
		})
		ui.Must(file.Write(filepath.Join(dirname, "main.tf")))

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.ARN(accountId, roles.DeployAdministrator),
		))
		networkAccount, err := cfg.FindSpecialAccount(ctx, accounts.Network)
		ui.Must(err)
		providersFile.Add(terraform.NetworkProviderFor(
			region,
			roles.ARN(aws.ToString(networkAccount.Id), roles.Auditor),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, cfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))

		ui.Must(terraform.Init(dirname))

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		ui.Must(err)
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Print("next, commit the following files to version control:")
	ui.Print("")
	ui.Print("modules/deploy/")
	ui.Print("root-modules/deploy/")
	ui.Print("")
	ui.Print("then, run `substrate create-admin-account`")
}
