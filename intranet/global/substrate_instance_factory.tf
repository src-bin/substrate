data "aws_iam_policy_document" "substrate-instance-factory" {
  statement {
    actions = [
      "autoscaling:DescribeAutoScalingGroups",
      "autoscaling:UpdateAutoScalingGroup",
      "ec2:DescribeInstances",
    ]
    resources = ["*"]
  }
}

module "substrate-instance-factory" {
  name   = "substrate-instance-factory"
  policy = data.aws_iam_policy_document.substrate-instance-factory.json
  source = "../../lambda-function/global"
}
