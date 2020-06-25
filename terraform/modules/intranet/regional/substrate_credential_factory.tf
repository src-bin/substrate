module "substrate-credential-factory" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-credential-factory.zip"
  name                     = "substrate-credential-factory"
  role_arn                 = var.substrate_credential_factory_role_arn
  source                   = "../../lambda-function/regional"
}
