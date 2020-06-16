package terraform

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awsdynamodb"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/ui"
)

//go:generate go run ../tools/template/main.go -name gitignoreTemplate gitignore.template
//go:generate go run ../tools/template/main.go -name makefileTemplate Makefile.template
//go:generate go run ../tools/template/main.go -name terraformBackendTemplate terraform.tf

// Root sets up the given directory as a root Terraform module by creating a
// few local files and AWS resources.
// - Makefile, a convenience for running Terraform from other directories.
// - .gitignore, to avoid committing providers and Lambda zip files.
// - terraform.tf, for configuring DynamoDB/S3-backed Terraform state files.
func Root(dirname string, sess *session.Session) error {
	if err := gitignore(dirname); err != nil {
		return err
	}
	if err := makefile(dirname); err != nil {
		return err
	}
	if err := terraformBackend(dirname, sess); err != nil {
		return err
	}
	return nil
}

func gitignore(dirname string) error {
	f, err := os.Create(path.Join(dirname, ".gitignore"))
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
	f, err := os.Create(path.Join(dirname, "Makefile"))
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

func terraformBackend(dirname string, sess *session.Session) error {
	f, err := os.Create(path.Join(dirname, "terraform.tf"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("terraform.tf").Parse(terraformBackendTemplate())
	if err != nil {
		return err
	}

	prefix := choices.Prefix()
	v := struct {
		Bucket, DynamoDBTable, Key, Region string
	}{
		Bucket:        fmt.Sprintf("%s-terraform-state", prefix),
		DynamoDBTable: fmt.Sprintf("%s-terraform-state-locks", prefix),
		Key:           dirname,
		Region:        choices.DefaultRegion(),
	}

	// Ensure the DynamoDB table and S3 bucket exist before configuring
	// Terraform to use them for remote state.
	ui.Spin("finding or creating an S3 bucket for storing Terraform state")
	callerIdentity, err := awssts.GetCallerIdentity(sts.New(sess))
	if err != nil {
		return err
	}
	org, err := awsorgs.DescribeOrganization(organizations.New(sess))
	if err != nil {
		return err
	}
	if err := awss3.EnsureBucket(
		s3.New(sess, &aws.Config{Region: aws.String(v.Region)}),
		v.Bucket,
		v.Region,
		&policies.Document{
			Statement: []policies.Statement{
				{
					Principal: &policies.Principal{AWS: []string{aws.StringValue(callerIdentity.Account)}},
					Action:    []string{"s3:*"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", v.Bucket),
						fmt.Sprintf("arn:aws:s3:::%s/*", v.Bucket),
					},
				},
				{
					Principal: &policies.Principal{AWS: []string{"*"}},
					Action:    []string{"s3:GetObject", "s3:ListBucket", "s3:PutObject"},
					Resource: []string{
						fmt.Sprintf("arn:aws:s3:::%s", v.Bucket),
						fmt.Sprintf("arn:aws:s3:::%s/*", v.Bucket),
					},
					Condition: policies.Condition{"StringEquals": {"aws:PrincipalOrgID": aws.StringValue(org.Id)}},
				},
			},
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
