module "substrate-okta-authenticator" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-okta-authenticator.zip"
  name                     = "substrate-okta-authenticator"
  role_arn                 = var.substrate_okta_authenticator_role_arn
  source                   = "../../lambda-function/regional"
}
