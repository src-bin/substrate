module "substrate-okta-authenticator" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-okta-authenticator.zip"
  name                     = "substrate-okta-authenticator"
  role_arn                 = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/substrate-okta-authenticator" # breaks a dependency cycle
  source                   = "../../lambda-function/regional"
}
