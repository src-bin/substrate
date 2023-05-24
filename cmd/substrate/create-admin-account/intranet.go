package createadminaccount

import (
	"context"
	_ "embed"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/policies"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
)

const (
	Intranet                     = "Intranet"
	IntranetAPIGateway           = "IntranetAPIGateway"
	IntranetAPIGatewayAuthorizer = "IntranetAPIGatewayAuthorizer"
)

//go:generate make -C ../../.. go-generate-intranet
//go:generate touch -t 202006100000.00 bootstrap
//go:generate zip -X substrate-intranet.zip bootstrap
//go:embed substrate-intranet.zip
var SubstrateIntranetZip []byte

// ensureIntranet manages all the API Gateway (v2), Certificate Manager, EC2,
// IAM, Lambda, Route 53, and Secrets Manager resources for the Intranet, no
// Terraform necessary (anymore). It must be called in an admin account.
func ensureIntranet(
	ctx context.Context,
	cfg *awscfg.Config,
	dnsDomainName string, // where the Intranet will be hosted (XXX initially on v2.${dnsDomainName} XXX)
	clientId, clientSecret string, // for all OAuth OIDC providers
	hostname string, // only for Okta
	tenantId string, // only for Azure AD
) error {
	var (
		err    error
		policy *awsiam.Policy
		role   *awsiam.Role
	)

	ui.Spin("provisioning your Intranet") // TODO do new spinners for Credential Factory and Instance Factory
	if policy, err = awsiam.EnsurePolicy(ctx, cfg, Intranet, &policies.Document{
		Statement: []policies.Statement{
			{
				Action: []string{
					"organizations:DescribeOrganization",
					"sts:AssumeRole",
				},
				Resource: []string{"*"},
				Sid:      "Accounts",
			},
			{
				Action: []string{
					"iam:CreateAccessKey",
					"iam:DeleteAccessKey",
					"iam:ListAccessKeys",
					"iam:ListUserTags",
					"iam:TagUser",
					"iam:UntagUser",
					"secretsmanager:CreateSecret",
					"secretsmanager:GetSecretValue",
					"secretsmanager:PutResourcePolicy",
					"secretsmanager:PutSecretValue",
					"secretsmanager:TagResource",
					"sts:AssumeRole",
				},
				Resource: []string{"*"},
				Sid:      "CredentialFactory",
			},
			{
				Action: []string{
					"apigateway:GET",
				},
				Resource: []string{"*"},
				Sid:      "Index",
			},
			{
				Action: []string{
					"ec2:CreateTags",
					"ec2:DescribeInstanceTypeOfferings",
					"ec2:DescribeInstanceTypes",
					"ec2:DescribeImages",
					"ec2:DescribeInstances",
					"ec2:DescribeKeyPairs",
					"ec2:DescribeLaunchTemplates",
					"ec2:DescribeLaunchTemplateVersions",
					"ec2:DescribeSecurityGroups",
					"ec2:DescribeSubnets",
					"ec2:ImportKeyPair",
					"ec2:RunInstances",
					"ec2:TerminateInstances",
					"iam:PassRole",
					"organizations:DescribeOrganization",
					"sts:AssumeRole",
				},
				Resource: []string{"*"},
				Sid:      "InstanceFactory",
			},
			{
				Action: []string{
					"secretsmanager:GetSecretValue",
				},
				Resource: []string{"*"},
				Sid:      "Login",
			},
			{
				Action: []string{
					"ec2:CreateNetworkInterface",
					"ec2:DeleteNetworkInterface",
					"ec2:DescribeNetworkInterfaces",
				},
				Resource: []string{"*"},
				Sid:      "Proxy",
			},
		},
	}); err != nil {
		return ui.StopErr(err)
	}
	if role, err = awsiam.EnsureRole(ctx, cfg, Intranet, policies.AssumeRolePolicyDocument(&policies.Principal{
		Service: []string{"lambda.amazonaws.com"},
	})); err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, role.Name, aws.ToString(policy.Arn)); err != nil {
		return ui.StopErr(err)
	}
	if policy, err = awsiam.EnsurePolicy(ctx, cfg, IntranetAPIGateway, &policies.Document{
		Statement: []policies.Statement{{
			Action:   []string{"lambda:InvokeFunction"},
			Resource: []string{"*"},
		}},
	}); err != nil {
		return ui.StopErr(err)
	}
	if role, err = awsiam.EnsureRole(ctx, cfg, IntranetAPIGateway, policies.AssumeRolePolicyDocument(&policies.Principal{
		Service: []string{"apigateway.amazonaws.com"},
	})); err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, role.Name, aws.ToString(policy.Arn)); err != nil {
		return ui.StopErr(err)
	}
	if policy, err = awsiam.EnsurePolicy(ctx, cfg, IntranetAPIGatewayAuthorizer, &policies.Document{
		Statement: []policies.Statement{{
			Action:   []string{"secretsmanager:GetSecretValue", "ec2:DescribeInstances"},
			Resource: []string{"*"},
		}},
	}); err != nil {
		return ui.StopErr(err)
	}
	if role, err = awsiam.EnsureRole(ctx, cfg, IntranetAPIGatewayAuthorizer, policies.AssumeRolePolicyDocument(&policies.Principal{
		Service: []string{"lambda.amazonaws.com"},
	})); err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.AttachRolePolicy(ctx, cfg, role.Name, aws.ToString(policy.Arn)); err != nil {
		return ui.StopErr(err)
	}
	for _, region := range regions.Selected() {
		_ = region //log.Printf("TODO ensureIntranet regional %s", region)
	}
	ui.Stop("ok")

	ui.Spin("configuring the Credential Factory to mint AWS credentials that last 12 hours")
	if policy, err = awsiam.EnsurePolicy(ctx, cfg, users.CredentialFactory, &policies.Document{
		Statement: []policies.Statement{
			{
				Action:   []string{"organizations:DescribeOrganization"},
				Resource: []string{"*"},
			},
			{
				Action:   []string{"sts:AssumeRole"},
				Resource: []string{"*"},
			},
		},
	}); err != nil {
		return ui.StopErr(err)
	}
	if err := awsiam.AttachUserPolicy(ctx, cfg, users.CredentialFactory, aws.ToString(policy.Arn)); err != nil {
		return ui.StopErr(err)
	}
	ui.Stop("ok")

	// The Administrator and Auditor roles  already exist but, since they
	// aren't managed by `substrate create-role`, they need EC2 instance
	// profiles in order for anyone assigned those roles to be able to use
	// the Instance Factory.
	ui.Spin("configuring the Instance Factory")
	if _, err := awsiam.EnsureInstanceProfile(ctx, cfg, roles.Administrator); err != nil {
		return ui.StopErr(err)
	}
	if _, err := awsiam.EnsureInstanceProfile(ctx, cfg, roles.Auditor); err != nil {
		return ui.StopErr(err)
	}
	for _, region := range regions.Selected() {
		_ = region //log.Printf("TODO ensureIntranet regional %s", region)
	}
	ui.Stop("ok")

	return nil
}
