data "aws_iam_policy_document" "okta-authenticator" {
  statement {
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

module "okta-authenticator" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/okta-authenticator.zip"
  name                     = "okta-authenticator"
  policy                   = data.aws_iam_policy_document.okta-authenticator.json
  source                   = "../lambda-function"
}
