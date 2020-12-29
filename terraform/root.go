package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsdynamodb"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/choices"
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
// state in the given region.
// - Makefile, a convenience for running Terraform from other directories.
// - .gitignore, to avoid committing providers and Lambda zip files.
// - terraform.tf, for configuring DynamoDB/S3-backed Terraform state files.
// TODO factor all the code generation of providers, the shared-between-accounts module for a domain, etc. into a RootModule type
func Root(dirname, region string) error {
	sess, err := awssessions.InSpecialAccount(
		accounts.Deploy,
		roles.DeployAdministrator,
		awssessions.Config{},
	)
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
	if err := terraformBackend(dirname, region, sess); err != nil {
		return err
	}
	return nil
}

func S3BucketName(region string) string {
	return fmt.Sprintf("%s-terraform-state-%s", choices.Prefix(), region)
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

func terraformBackend(dirname, region string, sess *session.Session) error {
	f, err := os.Create(filepath.Join(dirname, "terraform.tf"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("terraform.tf").Parse(terraformBackendTemplate())
	if err != nil {
		return err
	}

	deployAccount, err := awsorgs.FindSpecialAccount(
		organizations.New(awssessions.Must(awssessions.AssumeRoleManagement(
			sess,
			roles.OrganizationReader,
		))),
		accounts.Deploy,
	)
	if err != nil {
		return err
	}
	v := RemoteStateConfig{
		Bucket:        S3BucketName(region),
		DynamoDBTable: DynamoDBTableName,
		Key:           filepath.Join(dirname, "terraform.tfstate"),
		Region:        region,
		RoleArn:       roles.Arn(aws.StringValue(deployAccount.Id), roles.TerraformStateManager),
	}

	// Ensure the DynamoDB table and S3 bucket exist before configuring
	// Terraform to use them for remote state.
	ui.Spin("finding or creating an S3 bucket for storing Terraform state")
	callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		return err
	}
	if err := awss3.EnsureBucket(
		s3.New(sess, &aws.Config{Region: aws.String(v.Region)}),
		v.Bucket,
		v.Region,
		&policies.Document{
			Statement: []policies.Statement{{
				Principal: &policies.Principal{AWS: []string{aws.StringValue(callerIdentity.Account)}},
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
		dynamodb.New(sess, &aws.Config{Region: aws.String(v.Region)}),
		v.DynamoDBTable,
		[]*dynamodb.AttributeDefinition{&dynamodb.AttributeDefinition{
			AttributeName: aws.String("LockID"),
			AttributeType: aws.String("S"),
		}},
		[]*dynamodb.KeySchemaElement{&dynamodb.KeySchemaElement{
			AttributeName: aws.String("LockID"),
			KeyType:       aws.String("HASH"),
		}},
	); err != nil {
		return err
	}
	ui.Stopf("table %s", v.DynamoDBTable)

	return tmpl.Execute(f, v)
}
