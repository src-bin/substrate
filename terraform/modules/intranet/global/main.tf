data "aws_iam_policy_document" "apigateway" {
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

data "aws_iam_policy_document" "substrate-apigateway-authenticator" {
  statement {
    actions   = ["secretsmanager:GetSecretValue", "sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "substrate-apigateway-authorizer" {
  statement {
    actions   = ["secretsmanager:GetSecretValue", "sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "substrate-instance-factory" {
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

data "aws_iam_role" "admin" {
  name = "Administrator"
}

data "aws_route53_zone" "intranet" {
  name         = "${var.dns_domain_name}."
  private_zone = false
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

# Hoisted out of ../../lambda-function/global to allow logging while still
# running the Credential Factory directly as the Administrator role.
resource "aws_iam_role_policy_attachment" "admin-cloudwatch" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
  role       = data.aws_iam_role.admin.name
}

resource "aws_route53_record" "validation" {
  name    = aws_acm_certificate.intranet.domain_validation_options.0.resource_record_name
  records = [aws_acm_certificate.intranet.domain_validation_options.0.resource_record_value]
  ttl     = 60
  type    = aws_acm_certificate.intranet.domain_validation_options.0.resource_record_type
  zone_id = data.aws_route53_zone.intranet.zone_id
}