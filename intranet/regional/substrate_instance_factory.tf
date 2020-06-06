module "substrate-instance-factory" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-instance-factory.zip"
  name                     = "substrate-instance-factory"
  role_arn                 = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/substrate-instance-factory" # breaks a dependency cycle
  source                   = "../../lambda-function/regional"
}
