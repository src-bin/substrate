package terraform

// managed by go generate; do not edit by hand

func intranetRegionalTemplate() map[string]string {
	return map[string]string{
		"outputs.tf":                      `output "url" {
  value = "https://${aws_api_gateway_rest_api.intranet.id}.execute_api.${data.aws_region.current.name}.amazonaws.com/${var.stage_name}"
}
`,
		"substrate_okta_authenticator.tf": `module "substrate-okta-authenticator" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-okta-authenticator.zip"
  name                     = "substrate-okta-authenticator"
  role_arn                 = var.substrate_okta_authenticator_role_arn
  source                   = "../../lambda-function/regional"
}
`,
		"variables.tf":                    `variable "apigateway_role_arn" {}

variable "okta_client_id" {}

variable "okta_client_secret_timestamp" {} # TODO it's awkward to have to apply this Terraform in order to know how to set this

variable "okta_hostname" {}

variable "stage_name" {}

variable "substrate_instance_factory_role_arn" {}

variable "substrate_okta_authenticator_role_arn" {}

variable "substrate_okta_authorizer_role_arn" {}
`,
		"data.tf":                         `data "aws_caller_identity" "current" {}

data "aws_region" "current" {}
`,
		"apigateway.tf":                   `resource "aws_api_gateway_account" "current" {
  cloudwatch_role_arn = var.apigateway_role_arn
}

resource "aws_api_gateway_authorizer" "okta" {
  authorizer_credentials           = var.apigateway_role_arn
  authorizer_result_ttl_in_seconds = 1 # XXX longer once we know it's working; default 300
  authorizer_uri                   = module.substrate-okta-authorizer.invoke_arn
  identity_source                  = "method.request.header.Cookie"
  name                             = "Okta"
  rest_api_id                      = aws_api_gateway_rest_api.intranet.id
  type                             = "REQUEST"
}

resource "aws_api_gateway_deployment" "intranet" {
  lifecycle {
    create_before_destroy = true
  }
  rest_api_id = aws_api_gateway_rest_api.intranet.id
  stage_name  = var.stage_name
  triggers = {
    redeployment = sha1(join(",", list(
      jsonencode(aws_api_gateway_authorizer.okta),
      jsonencode(aws_api_gateway_integration.GET-instance-factory),
      jsonencode(aws_api_gateway_integration.GET-login),
      jsonencode(aws_api_gateway_integration.POST-login),
      jsonencode(aws_api_gateway_method.GET-instance-factory),
      jsonencode(aws_api_gateway_method.GET-login),
      jsonencode(aws_api_gateway_method.POST-login),
      jsonencode(aws_api_gateway_resource.instance-factory),
      jsonencode(aws_api_gateway_resource.login),
    )))
  }
  variables = {
    "OktaClientID"              = var.okta_client_id
    "OktaClientSecretTimestamp" = var.okta_client_secret_timestamp
    "OktaHostname"              = var.okta_hostname
  }
}

resource "aws_api_gateway_integration" "GET-instance-factory" {
  credentials             = var.apigateway_role_arn
  http_method             = aws_api_gateway_method.GET-instance-factory.http_method
  integration_http_method = "POST"
  passthrough_behavior    = "NEVER"
  resource_id             = aws_api_gateway_resource.instance-factory.id
  rest_api_id             = aws_api_gateway_rest_api.intranet.id
  type                    = "AWS_PROXY"
  uri                     = module.substrate-instance-factory.invoke_arn
}

resource "aws_api_gateway_integration" "GET-login" {
  credentials             = var.apigateway_role_arn
  http_method             = aws_api_gateway_method.GET-login.http_method
  integration_http_method = "POST"
  passthrough_behavior    = "NEVER"
  resource_id             = aws_api_gateway_resource.login.id
  rest_api_id             = aws_api_gateway_rest_api.intranet.id
  type                    = "AWS_PROXY"
  uri                     = module.substrate-okta-authenticator.invoke_arn
}

resource "aws_api_gateway_integration" "POST-login" {
  credentials             = var.apigateway_role_arn
  http_method             = aws_api_gateway_method.POST-login.http_method
  integration_http_method = "POST"
  passthrough_behavior    = "NEVER"
  resource_id             = aws_api_gateway_resource.login.id
  rest_api_id             = aws_api_gateway_rest_api.intranet.id
  type                    = "AWS_PROXY"
  uri                     = module.substrate-okta-authenticator.invoke_arn
}

resource "aws_api_gateway_method" "GET-instance-factory" {
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.okta.id
  http_method   = "GET"
  resource_id   = aws_api_gateway_resource.instance-factory.id
  rest_api_id   = aws_api_gateway_rest_api.intranet.id
}

resource "aws_api_gateway_method" "GET-login" {
  authorization = "NONE"
  http_method   = "GET"
  resource_id   = aws_api_gateway_resource.login.id
  rest_api_id   = aws_api_gateway_rest_api.intranet.id
}

resource "aws_api_gateway_method" "POST-login" {
  authorization = "NONE"
  http_method   = "POST"
  resource_id   = aws_api_gateway_resource.login.id
  rest_api_id   = aws_api_gateway_rest_api.intranet.id
}

resource "aws_api_gateway_method_settings" "intranet" {
  depends_on  = [aws_api_gateway_account.current]
  method_path = "*/*"
  rest_api_id = aws_api_gateway_rest_api.intranet.id
  settings {
    logging_level   = "INFO"
    metrics_enabled = false
  }
  stage_name = aws_api_gateway_deployment.intranet.stage_name
}

resource "aws_api_gateway_resource" "instance-factory" {
  parent_id   = aws_api_gateway_rest_api.intranet.root_resource_id
  path_part   = "instance-factory"
  rest_api_id = aws_api_gateway_rest_api.intranet.id
}

resource "aws_api_gateway_resource" "login" {
  parent_id   = aws_api_gateway_rest_api.intranet.root_resource_id
  path_part   = "login"
  rest_api_id = aws_api_gateway_rest_api.intranet.id
}

resource "aws_api_gateway_rest_api" "intranet" {
  endpoint_configuration {
    types = ["REGIONAL"]
  }
  name = "Intranet"
  tags = {
    Manager = "Terraform"
  }
}

resource "aws_cloudwatch_log_group" "apigateway" {
  name              = "API-Gateway-Execution-Logs_${aws_api_gateway_rest_api.intranet.id}/${aws_api_gateway_deployment.intranet.stage_name}"
  retention_in_days = 1
  tags = {
    Manager = "Terraform"
  }
}

resource "aws_cloudwatch_log_group" "apigateway-welcome" {
  name              = "/aws/apigateway/welcome"
  retention_in_days = 1
  tags = {
    Manager = "Terraform"
  }
}
`,
		"substrate_instance_factory.tf":   `module "substrate-instance-factory" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-instance-factory.zip"
  name                     = "substrate-instance-factory"
  role_arn                 = var.substrate_instance_factory_role_arn
  source                   = "../../lambda-function/regional"
}
`,
		"substrate_okta_authorizer.tf":    `module "substrate-okta-authorizer" {
  #apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  #apigateway_execution_arn = "arn:aws:apigateway:${data.aws_region.current.name}::*"
  #apigateway_execution_arn = "arn:aws:apigateway:${data.aws_region.current.name}::/restapis/${aws_api_gateway_rest_api.intranet.id}/authorizers/${aws_api_gateway_authorizer.okta.id}"
  apigateway_execution_arn = "arn:aws:execute-api:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:${aws_api_gateway_rest_api.intranet.id}/*"
  filename                 = "${path.module}/substrate-okta-authorizer.zip"
  name                     = "substrate-okta-authorizer"
  role_arn                 = var.substrate_okta_authorizer_role_arn
  source                   = "../../lambda-function/regional"
}
`,
	}
}
