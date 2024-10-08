package terraform

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsdynamodb"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awss3"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const DynamoDBTableName = "terraform-state-locks"

// EnsureStateManager manages an S3 bucket, a DynamoDB table, and an IAM role
// in the Substrate account that every other account in the organization can
// use to read, write, and lock Terraform state. This must be called in the
// Substrate account.
func EnsureStateManager(ctx context.Context, cfg *awscfg.Config) (*awsiam.Role, error) {
	//log.Print(jsonutil.MustString(cfg.MustGetCallerIdentity(ctx)))
	ui.Spin("finding or creating an S3 bucket, DynamoDB table, and IAM role for Terraform to use to manage remote state")

	// Gather up a list of principals that we expect to run Terraform so we
	// can allow them to assume the TerraformStateManager role.
	var terraformPrincipals []string
	allAccounts, err := cfg.ListAccounts(ctx)
	if err != nil {
		return nil, ui.StopErr(err)
	}
	for _, account := range allAccounts {
		if account.Tags[tagging.SubstrateSpecialAccount] == naming.Deploy {
			terraformPrincipals = append(terraformPrincipals, roles.ARN(aws.ToString(account.Id), roles.DeployAdministrator))
		} else if account.Tags[tagging.SubstrateSpecialAccount] == naming.Network {
			terraformPrincipals = append(terraformPrincipals, roles.ARN(aws.ToString(account.Id), roles.NetworkAdministrator))
		} else if account.Tags[tagging.SubstrateType] == naming.Substrate {
			terraformPrincipals = append(
				terraformPrincipals,
				roles.ARN(aws.ToString(account.Id), roles.Administrator),
				roles.ARN(aws.ToString(account.Id), roles.Substrate),
				users.ARN(aws.ToString(account.Id), users.Substrate),
			)
		} else if account.Tags[tagging.SubstrateSpecialAccount] == "management" || account.Tags[tagging.SubstrateType] == "management" {
			terraformPrincipals = append(
				terraformPrincipals,
				roles.ARN(aws.ToString(account.Id), roles.Substrate),
				users.ARN(aws.ToString(account.Id), users.Substrate),
			)
		} else if account.Tags[tagging.SubstrateType] == "service" || account.Tags[tagging.Domain] != "" {
			terraformPrincipals = append(terraformPrincipals, roles.ARN(aws.ToString(account.Id), roles.Administrator))
		}
	}
	//sort.Strings(terraformPrincipals) // to avoid spurious policy diffs
	//log.Printf("%+v", terraformPrincipals)

	// Create S3 buckets for storing Terraform state in every region we're
	// using. Gather resource strings for the TerraformStateManager policy
	// so that policy can be written in a least-privilege fashion, too.
	accountId, err := cfg.AccountId(ctx)
	if err != nil {
		return nil, ui.StopErr(err)
	}
	defaultAndSelectedRegions := []string{regions.Default()}
	for _, region := range regions.Selected() {
		if region != regions.Default() {
			defaultAndSelectedRegions = append(defaultAndSelectedRegions, region)
		}
	}
	var resources []string
	for _, region := range defaultAndSelectedRegions {

		bucketName := S3BucketName(region)
		ui.Spinf("creating the %s S3 bucket and %s DynamoDB table in %s", bucketName, DynamoDBTableName, region)
		statement := policies.Statement{

			// We're using policies attached to the TerraformStateManager
			// role to control access. This bucket policy is the standard
			// one that allows the owning account to control the bucket.
			Principal: &policies.Principal{AWS: []string{accountId}},

			Action: []string{"s3:*"},
			Resource: []string{
				fmt.Sprintf("arn:aws:s3:::%s", bucketName),
				fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
			},
		}
		if err := awss3.EnsureBucket(
			ctx,
			cfg.Regional(region),
			bucketName,
			region,
			&policies.Document{Statement: []policies.Statement{statement}},
		); awsutil.ErrorCodeIs(err, awss3.BucketAlreadyExists) {

			// Take the BucketAlreadyExists error (which is distinct from the
			// BucketAlreadyOwnedByYou error) as a sign that the bucket's in
			// the (legacy) deploy account. Switch to that account and do this
			// over again.
			ui.Stop("bucket already exists")
			ui.Stop("switching to the deploy account")
			cfg, err = cfg.AssumeSpecialRole(ctx, naming.Deploy, roles.DeployAdministrator, time.Hour)
			if err != nil {
				return nil, ui.StopErr(ui.StopErr(err))
			}
			return EnsureStateManager(ctx, cfg)

		} else if err != nil {
			return nil, ui.StopErr(ui.StopErr(err))
		}
		resources = append(resources, statement.Resource...)

		if _, err := awsdynamodb.EnsureTable(
			ctx,
			cfg.Regional(region),
			DynamoDBTableName,
			[]awsdynamodb.AttributeDefinition{{
				AttributeName: aws.String("LockID"),
				AttributeType: types.ScalarAttributeTypeS,
			}},
			[]awsdynamodb.KeySchemaElement{{
				AttributeName: aws.String("LockID"),
				KeyType:       types.KeyTypeHash,
			}},
		); err != nil {
			return nil, ui.StopErr(ui.StopErr(err))
		}

		ui.Stop("ok")
	}

	policy, err := awsiam.EnsurePolicy(
		ctx,
		cfg,
		roles.TerraformStateManager, // reuse role name as policy name
		&policies.Document{
			Statement: []policies.Statement{
				{
					Action: []string{"dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:DeleteItem"},
					Resource: []string{
						fmt.Sprintf("arn:aws:dynamodb:*:*:table/%s", DynamoDBTableName),
					},
				},
				{
					Action:   []string{"organizations:DescribeOrganization"}, // for assuming this role; see WaitUntilCredentialsWork
					Resource: []string{"*"},
				},
				{
					Action:   []string{"s3:DeleteObject", "s3:GetObject", "s3:ListBucket", "s3:PutObject"},
					Resource: resources,
				},
			},
		},
	)
	//log.Print(jsonutil.MustString(policies.AssumeRolePolicyDocument(&policies.Principal{AWS: terraformPrincipals})))
	role, err := awsiam.EnsureRole(
		ctx,
		cfg,
		roles.TerraformStateManager,
		policies.AssumeRolePolicyDocument(&policies.Principal{AWS: terraformPrincipals}),
	)
	if err != nil {
		return nil, ui.StopErr(err)
	}
	if err := awsiam.AttachRolePolicy(
		ctx,
		cfg,
		role.Name,
		aws.ToString(policy.Arn),
	); err != nil {
		return nil, ui.StopErr(err)
	}

	ui.Stopf("role %s", role.Name)
	//log.Printf("%+v", role)
	return role, nil
}

type RemoteState struct {
	Config   RemoteStateConfig
	Label    Value
	Provider ProviderAlias
}

type RemoteStateConfig struct {
	Bucket, DynamoDBTable, Key, Region, RoleArn string
}

func (rs RemoteState) Ref() Value {
	return Uf("data.terraform_remote_state.%s", rs.Label)
}

func (RemoteState) Template() string {
	return `data "terraform_remote_state" {{.Label.Value}} {
  backend = "s3"
  config = {
    bucket = "{{.Config.Bucket}}"
    dynamodb_table = "{{.Config.DynamoDBTable}}"
    key = "{{.Config.Key}}"
    region = "{{.Config.Region}}"
    role_arn = "{{.Config.RoleArn}}"
  }
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
}`
}
