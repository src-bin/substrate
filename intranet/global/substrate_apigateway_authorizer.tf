data "aws_iam_policy_document" "substrate-apigateway-authorizer" {
  statement {
    actions   = ["secretsmanager:GetSecretValue", "sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

module "substrate-apigateway-authorizer" {
  name   = "substrate-apigateway-authorizer"
  policy = data.aws_iam_policy_document.substrate-apigateway-authorizer.json
  source = "../../lambda-function/global"
}
