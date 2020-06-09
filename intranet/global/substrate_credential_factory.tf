data "aws_iam_policy_document" "substrate-credential-factory" {
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
