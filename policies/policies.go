package policies

const (
	AllowAssumeRoleName    = "SubstrateAllowAssumeRole"
	DenySensitiveReadsName = "SubstrateDenySensitiveReads"
)

var (
	AllowAssumeRole = &Document{
		Statement: []Statement{{
			Action:   []string{"sts:AssumeRole"},
			Effect:   Allow,
			Resource: []string{"*"},
		}},
	}
	DenySensitiveReads = &Document{ // <https://alestic.com/2015/10/aws-iam-readonly-too-permissive/>
		Statement: []Statement{{
			Action: []string{
				"cloudformation:GetTemplate", // note this is in conflict with Vanta's requested permissions
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
			Effect:   Deny,
			Resource: []string{"*"},
		}},
	}
)
