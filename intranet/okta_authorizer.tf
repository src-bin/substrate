data "aws_iam_policy_document" "okta-authorizer" {
  statement {
    actions   = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

data "aws_region" "current" {}

module "okta-authorizer" {
  #apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  #apigateway_execution_arn = "arn:aws:apigateway:${data.aws_region.current.name}::*"
  apigateway_execution_arn = "arn:aws:apigateway:${data.aws_region.current.name}::/restapis/${aws_api_gateway_rest_api.intranet.id}/authorizers/${aws_api_gateway_authorizer.okta.id}"
  filename                 = "${path.module}/okta-authorizer.zip"
  name                     = "okta-authorizer"
  policy                   = data.aws_iam_policy_document.okta-authorizer.json
  source                   = "../lambda-function"
}
