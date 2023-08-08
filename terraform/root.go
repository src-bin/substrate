package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsdynamodb"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

//go:generate go run ../tools/template/main.go -name gitignoreTemplate gitignore.template
//go:generate go run ../tools/template/main.go -name makefileTemplate Makefile.template
//go:generate go run ../tools/template/main.go -name terraformBackendTemplate terraform.tf

const DynamoDBTableName = "terraform-state-locks"

// Root sets up the given directory as a root Terraform module by creating a
// few local files and AWS resources.  Set it up to store remote Terraform
// state in the given region. It can only be called with a *Config with the
// Administrator role in an admin account or one already in the management
// account. It creates the following files:
// - Makefile, a convenience for running Terraform from other directories.
// - .gitignore, to avoid committing providers and Lambda zip files.
// - terraform.tf, for configuring DynamoDB/S3-backed Terraform state files.
// TODO factor all the code generation of providers, the shared-between-accounts module for a domain, etc. into a RootModule type
func Root(ctx context.Context, cfg *awscfg.Config, dirname, region string) (err error) {

	// Originally, we stored Terraform state from all accounts in the special
	// deploy account but, in an effort to simplify and streamline Substrate,
	// new installations won't actually have a special deploy account and
	// instead will store Terraform state in the Substrate account. In order
	// to accommodate both, we first check for the existence of a special
	// deploy account and use it if we can and otherwise fall back to the
	// Substrate account.
	stateCfg, err := cfg.AssumeSpecialRole(
		ctx,
		accounts.Deploy,
		roles.DeployAdministrator,
		time.Hour,
	)
	if err != nil {
		stateCfg, err = cfg.AssumeSubstrateRole(ctx, roles.Administrator, time.Hour)
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dirname, 0777); err != nil {
		return err
	}
	if err := gitignore(dirname); err != nil {
		return err
	}
	if err := makefile(dirname); err != nil {
		return err
	}
	if err := terraformBackend(ctx, stateCfg, dirname, region); err != nil {
		return err
	}
	if err := versions(dirname, nil, true); err != nil {
		return err
	}
	/*
		if err := Upgrade(dirname); err != nil {
			return err
		}
	*/
	return nil
}

func S3BucketName(region string) string {
	return fmt.Sprintf("%s-terraform-state-%s", naming.Prefix(), region)
}

func gitignore(dirname string) error {
	f, err := os.Create(filepath.Join(dirname, ".gitignore"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New(".gitignore").Parse(gitignoreTemplate())
	if err != nil {
		return err
	}
	return tmpl.Execute(f, nil)
}

// TODO consider deleting this altogether in favor of `terraform -chdir=... ...`
func makefile(dirname string) error {
	f, err := os.Create(filepath.Join(dirname, "Makefile"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("Makefile").Parse(makefileTemplate())
	if err != nil {
		return err
	}

	// Find out what GOBIN should be set to so that callers don't have to worry
	// about setting it themselves.
	pathname, err := os.Executable()
	if err != nil {
		return err
	}

	return tmpl.Execute(f, struct{ GOBIN string }{filepath.Dir(pathname)})
}

// terraformBackend creates the S3 bucket and DynamoDB table we use to store
// Terraform state for this root module. The given cfg must be in the Substrate
// account or, for older installations, the special deploy account.
func terraformBackend(
	ctx context.Context,
	cfg *awscfg.Config,
	dirname, region string,
) error {
	callerIdentity, err := cfg.GetCallerIdentity(ctx)
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dirname, "terraform.tf"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("terraform.tf").Parse(terraformBackendTemplate())
	if err != nil {
		return err
	}

	v := RemoteStateConfig{
		Bucket:        S3BucketName(region),
		DynamoDBTable: DynamoDBTableName,
		Key:           filepath.Join(dirname, "terraform.tfstate"),
		Region:        region,
		RoleArn: roles.ARN(
			aws.ToString(callerIdentity.Account),
			roles.TerraformStateManager,
		),
	}

	// Ensure the DynamoDB table and S3 bucket exist before configuring
	// Terraform to use them for remote state.
	ui.Spin("finding or creating an S3 bucket for storing Terraform state")
	if err := awss3.EnsureBucket(
		ctx,
		cfg.Regional(v.Region),
		v.Bucket,
		v.Region,
		&policies.Document{
			Statement: []policies.Statement{{
				Principal: &policies.Principal{AWS: []string{aws.ToString(callerIdentity.Account)}},
				Action:    []string{"s3:*"},
				Resource: []string{
					fmt.Sprintf("arn:aws:s3:::%s", v.Bucket),
					fmt.Sprintf("arn:aws:s3:::%s/*", v.Bucket),
				},
			}},
		},
	); err != nil {
		return err
	}
	ui.Stopf("bucket %s", v.Bucket)
	ui.Spin("finding or creating a DynamoDB table for Terraform state locking")
	if _, err := awsdynamodb.EnsureTable(
		ctx,
		cfg.Regional(v.Region),
		v.DynamoDBTable,
		[]awsdynamodb.AttributeDefinition{{
			AttributeName: aws.String("LockID"),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		[]awsdynamodb.KeySchemaElement{{
			AttributeName: aws.String("LockID"),
			KeyType:       types.KeyTypeHash,
		}},
	); err != nil {
		return err
	}
	ui.Stopf("table %s", v.DynamoDBTable)

	return tmpl.Execute(f, v)
}
