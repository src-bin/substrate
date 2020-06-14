module "substrate-apigateway-authenticator" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-apigateway-authenticator.zip"
  name                     = "substrate-apigateway-authenticator"
  role_arn                 = var.substrate_apigateway_authenticator_role_arn
  source                   = "../../lambda-function/regional"
}
