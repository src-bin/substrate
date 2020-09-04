package terraform

// managed by go generate; do not edit by hand

func intranetGlobalTemplate() map[string]string {
	return map[string]string{
		"main.tf":      `data "aws_iam_policy_document" "apigateway" {
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

data "aws_iam_policy_document" "credential-factory" {
  statement {
    actions   = ["sts:AssumeRole"]
    resources = [data.aws_iam_role.admin.arn]
  }
}

data "aws_iam_policy_document" "substrate-apigateway-authenticator" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "substrate-apigateway-authorizer" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "substrate-credential-factory" {
  statement {
    actions = [
      "iam:CreateAccessKey",
      "iam:DeleteAccessKey",
      "iam:ListAccessKeys",
    ]
    resources = ["*"]
  }
  statement {
    actions   = ["sts:AssumeRole"]
    resources = [data.aws_iam_role.admin.arn]
  }
}

data "aws_iam_policy_document" "substrate-instance-factory" {
  statement {
    actions = [
      "ec2:CreateTags",
      "ec2:DescribeInstanceTypeOfferings",
      "ec2:DescribeImages",
      "ec2:DescribeInstances",
      "ec2:DescribeKeyPairs",
      "ec2:DescribeSecurityGroups",
      "ec2:DescribeSubnets",
      "ec2:ImportKeyPair",
      "ec2:RunInstances",
      "ec2:TerminateInstances",
      "organizations:DescribeOrganization",
      "sts:AssumeRole",
    ]
    resources = ["*"]
  }
  statement {
    actions   = ["iam:PassRole"]
    resources = [data.aws_iam_role.admin.arn]
  }
}

data "aws_iam_role" "admin" {
  name = "Administrator"
}

data "aws_route53_zone" "intranet" {
  name         = "${var.dns_domain_name}."
  private_zone = false
}

data "aws_iam_user" "credential-factory" {
  user_name = "CredentialFactory"
}

module "substrate-apigateway-authenticator" {
  name   = "substrate-apigateway-authenticator"
  policy = data.aws_iam_policy_document.substrate-apigateway-authenticator.json
  source = "../../lambda-function/global"
}

module "substrate-apigateway-authorizer" {
  name   = "substrate-apigateway-authorizer"
  policy = data.aws_iam_policy_document.substrate-apigateway-authorizer.json
  source = "../../lambda-function/global"
}

module "substrate-credential-factory" {
  name   = "substrate-credential-factory"
  policy = data.aws_iam_policy_document.substrate-credential-factory.json
  source = "../../lambda-function/global"
}

module "substrate-instance-factory" {
  name   = "substrate-instance-factory"
  policy = data.aws_iam_policy_document.substrate-instance-factory.json
  source = "../../lambda-function/global"
}

resource "aws_acm_certificate" "intranet" {
  domain_name       = var.dns_domain_name
  validation_method = "DNS"
}

resource "aws_acm_certificate_validation" "intranet" {
  certificate_arn         = aws_acm_certificate.intranet.arn
  validation_record_fqdns = [aws_route53_record.validation.fqdn]
}

resource "aws_iam_instance_profile" "admin" {
  name = "Administrator"
  role = data.aws_iam_role.admin.name
}

resource "aws_iam_policy" "apigateway" {
  name   = "IntranetAPIGateway"
  policy = data.aws_iam_policy_document.apigateway.json
}

resource "aws_iam_policy" "credential-factory" {
  name   = "CredentialFactory"
  policy = data.aws_iam_policy_document.credential-factory.json
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

# Hoisted out of ../../lambda-function/global to allow logging while still
# running the Credential Factory directly as the Administrator role.
# TODO we're not doing this anymore so I think this can be removed.
resource "aws_iam_role_policy_attachment" "admin-cloudwatch" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
  role       = data.aws_iam_role.admin.name
}

resource "aws_iam_user_policy_attachment" "credential-factory" {
  policy_arn = aws_iam_policy.credential-factory.arn
  user       = data.aws_iam_user.credential-factory.user_name
}

resource "aws_route53_record" "validation" {
  name    = aws_acm_certificate.intranet.domain_validation_options.0.resource_record_name
  records = [aws_acm_certificate.intranet.domain_validation_options.0.resource_record_value]
  ttl     = 60
  type    = aws_acm_certificate.intranet.domain_validation_options.0.resource_record_type
  zone_id = data.aws_route53_zone.intranet.zone_id
}
`,
		"outputs.tf":   `output "apigateway_role_arn" {
  value = aws_iam_role.apigateway.arn
}

# substrate_credential_factory_role_arn is set to Administrator because there's
# a chicken-and-egg problem if we try to authorize a role specific to the
# substrate-credential-factory Lambda function to assume the Administrator role
# since its ARN is not known when the Administrator's assume role policy must
# be set.  And, since the whole point is to assume the Administrator role, it's
# no serious security compromise to jump straight to the Administrator role.
output "substrate_credential_factory_role_arn" {
  value = data.aws_iam_role.admin.arn
}

output "substrate_instance_factory_role_arn" {
  value = module.substrate-instance-factory.role_arn
}

output "substrate_apigateway_authenticator_role_arn" {
  value = module.substrate-apigateway-authenticator.role_arn
}

output "substrate_apigateway_authorizer_role_arn" {
  value = module.substrate-apigateway-authorizer.role_arn
}

output "validation_fqdn" {
  value = aws_route53_record.validation.fqdn
}
`,
		"substrate.tf": `module "substrate" {
  source = "../../substrate/global"
}
`,
		"variables.tf": `variable "dns_domain_name" {
  type = string
}
`,
	}
}
