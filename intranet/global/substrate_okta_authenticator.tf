data "aws_iam_policy_document" "substrate-okta-authenticator" {
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
