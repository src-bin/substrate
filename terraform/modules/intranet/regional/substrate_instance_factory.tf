module "substrate-instance-factory" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-instance-factory.zip"
  name                     = "substrate-instance-factory"
  role_arn                 = var.substrate_instance_factory_role_arn
  source                   = "../../lambda-function/regional"
}
