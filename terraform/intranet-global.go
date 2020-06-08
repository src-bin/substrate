package terraform

// managed by go generate; do not edit by hand

func intranetGlobalTemplate() map[string]string {
	return map[string]string{
		"substrate_okta_authenticator.tf": `data "aws_iam_policy_document" "substrate-okta-authenticator" {
  statement {
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

module "substrate-okta-authenticator" {
  name   = "substrate-okta-authenticator"
  policy = data.aws_iam_policy_document.substrate-okta-authenticator.json
  source = "../../lambda-function/global"
}
`,
		"substrate_okta_authorizer.tf":    `data "aws_iam_policy_document" "substrate-okta-authorizer" {
  statement {
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

module "substrate-okta-authorizer" {
  name   = "substrate-okta-authorizer"
  policy = data.aws_iam_policy_document.substrate-okta-authorizer.json
  source = "../../lambda-function/global"
}
`,
		"outputs.tf":                      `output "apigateway_role_arn" {
  value = aws_iam_role.apigateway.arn
}

output "substrate_instance_factory_role_arn" {
  value = module.substrate-instance-factory.role_arn
}

output "substrate_okta_authenticator_role_arn" {
  value = module.substrate-okta-authenticator.role_arn
}
output "substrate_okta_authorizer_role_arn" {
  value = module.substrate-okta-authorizer.role_arn
}
`,
		"iam.tf":                          `data "aws_iam_policy_document" "apigateway" {
  statement {
    actions   = ["lambda:InvokeFunction"]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "apigateway-trust" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["apigateway.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_policy" "apigateway" {
  name   = "IntranetAPIGateway"
  policy = data.aws_iam_policy_document.apigateway.json
}

resource "aws_iam_role" "apigateway" {
  assume_role_policy = data.aws_iam_policy_document.apigateway-trust.json
  name               = "IntranetAPIGateway"
}

resource "aws_iam_role_policy_attachment" "apigateway" {
  policy_arn = aws_iam_policy.apigateway.arn
  role       = aws_iam_role.apigateway.name
}

resource "aws_iam_role_policy_attachment" "apigateway-cloudwatch" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
  role       = aws_iam_role.apigateway.name
}
`,
		"variables.tf":                    `
`,
		"substrate_instance_factory.tf":   `data "aws_iam_policy_document" "substrate-instance-factory" {
  statement {
    actions = [
      "ec2:DescribeInstanceTypeOfferings",
      "ec2:DescribeImages",
      "ec2:DescribeInstances",
      "ec2:DescribeSubnets",
      "ec2:RunInstances",
      "ec2:TerminateInstances",
    ]
    resources = ["*"]
  }
}

module "substrate-instance-factory" {
  name   = "substrate-instance-factory"
  policy = data.aws_iam_policy_document.substrate-instance-factory.json
  source = "../../lambda-function/global"
}
`,
	}
}
