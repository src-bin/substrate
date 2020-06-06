output "substrate_instance_factory_function_arn" {
  value = module.substrate-instance-factory.function_arn
}

output "substrate_okta_authenticator_function_arn" {
  value = module.substrate-okta-authenticator.function_arn
}

output "substrate_okta_authorizer_function_arn" {
  value = module.substrate-okta-authorizer.function_arn
}


output "url" {
  value = "https://${aws_api_gateway_rest_api.intranet.id}.execute_api.${data.aws_region.current.name}.amazonaws.com/${var.stage_name}"
}
