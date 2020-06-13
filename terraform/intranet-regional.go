package terraform

// managed by go generate; do not edit by hand

func intranetRegionalTemplate() map[string]string {
	return map[string]string{
		"variables.tf":                    `variable "apigateway_role_arn" {
  type = string
}

variable "dns_domain_name" {
  type = string
}

variable "oauth_oidc_client_id" {
  type = string
}

variable "oauth_oidc_client_secret_timestamp" {
  type = string
}

variable "okta_hostname" {
  default = ""
  type    = string
}

variable "selected_regions" {
  type = list(string)
}

variable "stage_name" {
  type = string
}

variable "substrate_credential_factory_role_arn" {
  type = string
}

variable "substrate_instance_factory_role_arn" {
  type = string
}

variable "substrate_okta_authenticator_role_arn" {
  type = string
}

variable "substrate_okta_authorizer_role_arn" {
  type = string
}

variable "validation_fqdn" {
  type = string
}
`,
		"substrate_credential_factory.tf": `module "substrate-credential-factory" {
  apigateway_execution_arn = "${aws_api_gateway_deployment.intranet.execution_arn}/*"
  filename                 = "${path.module}/substrate-credential-factory.zip"
  name                     = "substrate-credential-factory"
  role_arn                 = var.substrate_credential_factory_role_arn
  source                   = "../../lambda-function/regional"
}
`,
		"route53.tf":                      `data "aws_route53_zone" "intranet" {
  name         = "${var.dns_domain_name}."
  private_zone = false
}

resource "aws_route53_record" "intranet" {
  alias {
    evaluate_target_health = true
    name                   = aws_api_gateway_domain_name.intranet.regional_domain_name
    zone_id                = aws_api_gateway_domain_name.intranet.regional_zone_id
  }
  latency_routing_policy {
    region = data.aws_region.current.name
  }
  name           = aws_api_gateway_domain_name.intranet.domain_name
  set_identifier = data.aws_region.current.name
  type           = "A"
  zone_id        = data.aws_route53_zone.intranet.id
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
		"outputs.tf":                      `
`,
		"apigateway.tf":                   `resource "aws_api_gateway_account" "current" {
  cloudwatch_role_arn = var.apigateway_role_arn
}

resource "aws_api_gateway_authorizer" "okta" {
  authorizer_credentials           = var.apigateway_role_arn
  authorizer_result_ttl_in_seconds = 1 # TODO longer once we know it's working; default 300
  authorizer_uri                   = module.substrate-okta-authorizer.invoke_arn
  identity_source                  = "method.request.header.Cookie"
  name                             = "Okta"
  rest_api_id                      = aws_api_gateway_rest_api.intranet.id
  type                             = "REQUEST"
}

resource "aws_api_gateway_base_path_mapping" "intranet" {
  api_id      = aws_api_gateway_rest_api.intranet.id
  stage_name  = aws_api_gateway_deployment.intranet.stage_name
  domain_name = aws_api_gateway_domain_name.intranet.domain_name
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
      jsonencode(aws_api_gateway_integration.GET-credential-factory),
      jsonencode(aws_api_gateway_integration.GET-instance-factory),
      jsonencode(aws_api_gateway_integration.GET-login),
      jsonencode(aws_api_gateway_integration.POST-credential-factory),
      jsonencode(aws_api_gateway_integration.POST-instance-factory),
      jsonencode(aws_api_gateway_integration.POST-login),
      jsonencode(aws_api_gateway_method.GET-credential-factory),
      jsonencode(aws_api_gateway_method.GET-instance-factory),
      jsonencode(aws_api_gateway_method.GET-login),
      jsonencode(aws_api_gateway_method.POST-credential-factory),
      jsonencode(aws_api_gateway_method.POST-instance-factory),
      jsonencode(aws_api_gateway_method.POST-login),
      jsonencode(aws_api_gateway_resource.credential-factory),
      jsonencode(aws_api_gateway_resource.instance-factory),
      jsonencode(aws_api_gateway_resource.login),
    )))
  }
  variables = {
    "OAuthOIDCClientID"              = var.oauth_oidc_client_id
    "OAuthOIDCClientSecretTimestamp" = var.oauth_oidc_client_secret_timestamp
    "OktaHostname"                   = var.okta_hostname
    "SelectedRegions"                = join(",", var.selected_regions)
  }
}

resource "aws_api_gateway_domain_name" "intranet" {
  domain_name = var.dns_domain_name
  endpoint_configuration {
    types = ["REGIONAL"]
  }
  regional_certificate_arn = aws_acm_certificate_validation.intranet.certificate_arn
  security_policy          = "TLS_1_2"
}

resource "aws_api_gateway_integration" "GET-credential-factory" {
  credentials             = var.apigateway_role_arn
  http_method             = aws_api_gateway_method.GET-credential-factory.http_method
  integration_http_method = "POST"
  passthrough_behavior    = "NEVER"
  resource_id             = aws_api_gateway_resource.credential-factory.id
  rest_api_id             = aws_api_gateway_rest_api.intranet.id
  type                    = "AWS_PROXY"
  uri                     = module.substrate-credential-factory.invoke_arn
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

resource "aws_api_gateway_integration" "POST-credential-factory" {
  credentials             = var.apigateway_role_arn
  http_method             = aws_api_gateway_method.POST-credential-factory.http_method
  integration_http_method = "POST"
  passthrough_behavior    = "NEVER"
  resource_id             = aws_api_gateway_resource.credential-factory.id
  rest_api_id             = aws_api_gateway_rest_api.intranet.id
  type                    = "AWS_PROXY"
  uri                     = module.substrate-credential-factory.invoke_arn
}

resource "aws_api_gateway_integration" "POST-instance-factory" {
  credentials             = var.apigateway_role_arn
  http_method             = aws_api_gateway_method.POST-instance-factory.http_method
  integration_http_method = "POST"
  passthrough_behavior    = "NEVER"
  resource_id             = aws_api_gateway_resource.instance-factory.id
  rest_api_id             = aws_api_gateway_rest_api.intranet.id
  type                    = "AWS_PROXY"
  uri                     = module.substrate-instance-factory.invoke_arn
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

resource "aws_api_gateway_method" "GET-credential-factory" {
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.okta.id
  http_method   = "GET"
  resource_id   = aws_api_gateway_resource.credential-factory.id
  rest_api_id   = aws_api_gateway_rest_api.intranet.id
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

resource "aws_api_gateway_method" "POST-credential-factory" {
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.okta.id
  http_method   = "POST"
  resource_id   = aws_api_gateway_resource.credential-factory.id
  rest_api_id   = aws_api_gateway_rest_api.intranet.id
}

resource "aws_api_gateway_method" "POST-instance-factory" {
  authorization = "CUSTOM"
  authorizer_id = aws_api_gateway_authorizer.okta.id
  http_method   = "POST"
  resource_id   = aws_api_gateway_resource.instance-factory.id
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

resource "aws_api_gateway_resource" "credential-factory" {
  parent_id   = aws_api_gateway_rest_api.intranet.root_resource_id
  path_part   = "credential-factory"
  rest_api_id = aws_api_gateway_rest_api.intranet.id
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
		"acm.tf":                          `resource "aws_acm_certificate" "intranet" {
  domain_name       = var.dns_domain_name
  validation_method = "DNS"
}

resource "aws_acm_certificate_validation" "intranet" {
  certificate_arn = aws_acm_certificate.intranet.arn
  #validation_record_fqdns = [aws_route53_record.validation.fqdn]
  validation_record_fqdns = [var.validation_fqdn]
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
		"data.tf":                         `data "aws_caller_identity" "current" {}

data "aws_region" "current" {}
`,
	}
}
