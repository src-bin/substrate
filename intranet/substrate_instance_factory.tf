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
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-instance-factory.zip"
  name                     = "substrate-instance-factory"
  policy                   = data.aws_iam_policy_document.substrate-instance-factory.json
  source                   = "../lambda-function"
}
