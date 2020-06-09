package terraform

// managed by go generate; do not edit by hand

func intranetGlobalTemplate() map[string]string {
	return map[string]string{
		"variables.tf":                    `variable "dns_domain_name" {
  type = string
}
`,
		"route53.tf":                      `data "aws_route53_zone" "intranet" {
  name         = "${var.dns_domain_name}."
  private_zone = false
}

resource "aws_route53_record" "validation" {
  name    = aws_acm_certificate.intranet.domain_validation_options.0.resource_record_name
  records = [aws_acm_certificate.intranet.domain_validation_options.0.resource_record_value]
  ttl     = 60
  type    = aws_acm_certificate.intranet.domain_validation_options.0.resource_record_type
  zone_id = data.aws_route53_zone.intranet.zone_id
}
`,
		"outputs.tf":                      `output "apigateway_role_arn" {
  value = aws_iam_role.apigateway.arn
}

output "substrate_credential_factory_role_arn" {
  value = module.substrate-credential-factory.role_arn
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

output "validation_fqdn" {
  value = aws_route53_record.validation.fqdn
}
`,
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
		"substrate_credential_factory.tf": `data "aws_iam_policy_document" "substrate-credential-factory" {
  statement {
    actions = [
      "sts:AssumeRole",
    ]
    resources = [data.aws_iam_role.admin.arn]
  }
}

data "aws_iam_role" "admin" {
  name = "Administrator"
}

module "substrate-credential-factory" {
  name   = "substrate-credential-factory"
  policy = data.aws_iam_policy_document.substrate-credential-factory.json
  source = "../../lambda-function/global"
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
		"acm.tf":                          `resource "aws_acm_certificate" "intranet" {
  domain_name       = var.dns_domain_name
  validation_method = "DNS"
}

resource "aws_acm_certificate_validation" "intranet" {
  certificate_arn         = aws_acm_certificate.intranet.arn
  validation_record_fqdns = [aws_route53_record.validation.fqdn]
}
`,
	}
}
