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

module "substrate-instance-factory" {
  name   = "substrate-instance-factory"
  policy = data.aws_iam_policy_document.substrate-instance-factory.json
  source = "../../lambda-function/global"
}
