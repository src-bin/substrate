package admin

import "github.com/src-bin/substrate/policies"

var (
	AllowAssumeRolePolicyDocument = &policies.Document{
		Statement: []policies.Statement{{
			Action:   []string{"sts:AssumeRole"},
			Effect:   policies.Allow,
			Resource: []string{"*"},
		}},
	}

	// <https://alestic.com/2015/10/aws-iam-readonly-too-permissive/>
	DenySensitiveReadsPolicyDocument = &policies.Document{
		Statement: []policies.Statement{{
			Action: []string{
				"cloudformation:GetTemplate", // TODO this is in conflict with Vanta's requested permissions
				"dynamodb:BatchGetItem",
				"dynamodb:GetItem",
				"dynamodb:Query",
				"dynamodb:Scan",
				"ec2:GetConsoleOutput",
				"ec2:GetConsoleScreenshot",
				"ecr:BatchGetImage",
				"ecr:GetAuthorizationToken",
				"ecr:GetDownloadUrlForLayer",
				"kinesis:Get*",
				"lambda:GetFunction",
				"logs:GetLogEvents",
				"s3:GetObject",
				"s3:GetObjectVersion", // believed to be redundant but best not to chance it
				"sdb:Select*",
				"sqs:ReceiveMessage",
			},
			Effect:   policies.Deny,
			Resource: []string{"*"},
		}},
	}
)
