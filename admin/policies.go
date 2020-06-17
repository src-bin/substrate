package admin

import "github.com/src-bin/substrate/policies"

// <https://alestic.com/2015/10/aws-iam-readonly-too-permissive/>
var DenySensitiveReadsPolicyDocument = &policies.Document{
	Statement: []policies.Statement{{
		Action: []string{
			"cloudformation:GetTemplate",
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
			"sdb:Select*",
			"sqs:ReceiveMessage",
		},
		Effect:   policies.Deny,
		Resource: []string{"*"},
	}},
}
