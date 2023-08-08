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

data "aws_iam_policy_document" "intranet" {
  statement {
    actions = [
      "organizations:DescribeOrganization",
      "sts:AssumeRole",
    ]
    resources = ["*"]
    sid       = "Accounts"
  }
  statement {
    actions = [
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
    ]
    resources = ["*"]
    sid       = "CredentialFactory"
  }
  statement {
    actions   = ["apigateway:GET"]
    resources = ["*"]
    sid       = "Index"
  }
  statement {
    actions = [
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
    ]
    resources = ["*"]
    sid       = "InstanceFactory"
  }
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = ["*"]
    sid       = "Login"
  }
  statement {
    actions = [
      "ec2:CreateNetworkInterface",
      "ec2:DeleteNetworkInterface",
      "ec2:DescribeNetworkInterfaces",
    ]
    resources = ["*"]
    sid       = "Proxy"
  }
}

data "aws_iam_policy_document" "intranet-apigateway-authorizer" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
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

module "intranet-apigateway-authorizer" {
  name   = "IntranetAPIGatewayAuthorizer"
  policy = data.aws_iam_policy_document.intranet-apigateway-authorizer.json
  source = "../../lambda-function/global"
}

module "intranet" {
  name   = "Intranet"
  policy = data.aws_iam_policy_document.intranet.json
  source = "../../lambda-function/global"
}

resource "aws_iam_instance_profile" "admin" {
  name = "Administrator"
  role = data.aws_iam_role.admin.name
}

resource "aws_iam_policy" "apigateway" {
  name   = "IntranetAPIGateway"
  policy = data.aws_iam_policy_document.apigateway.json
}

resource "aws_iam_role" "apigateway" {
  assume_role_policy   = data.aws_iam_policy_document.apigateway-trust.json
  max_session_duration = 43200
  name                 = "IntranetAPIGateway"
}

resource "aws_iam_role_policy_attachment" "apigateway" {
  policy_arn = aws_iam_policy.apigateway.arn
  role       = aws_iam_role.apigateway.name
}

resource "aws_iam_role_policy_attachment" "apigateway-cloudwatch" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
  role       = aws_iam_role.apigateway.name
}
