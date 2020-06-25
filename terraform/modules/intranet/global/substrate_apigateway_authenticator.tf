data "aws_iam_policy_document" "substrate-apigateway-authenticator" {
  statement {
    actions   = ["secretsmanager:GetSecretValue", "sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

module "substrate-apigateway-authenticator" {
  name   = "substrate-apigateway-authenticator"
  policy = data.aws_iam_policy_document.substrate-apigateway-authenticator.json
  source = "../../lambda-function/global"
}
