data "aws_iam_policy_document" "substrate-okta-authorizer" {
  statement {
    actions   = ["secretsmanager:GetSecretValue", "sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

module "substrate-okta-authorizer" {
  name   = "substrate-okta-authorizer"
  policy = data.aws_iam_policy_document.substrate-okta-authorizer.json
  source = "../../lambda-function/global"
}
