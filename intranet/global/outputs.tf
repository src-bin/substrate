output "apigateway_role_arn" {
  value = aws_iam_role.apigateway.arn
}

output "substrate_credential_factory_role_arn" {
  value = data.aws_iam_role.admin.arn
}

output "substrate_instance_factory_role_arn" {
  value = module.substrate-instance-factory.role_arn
}

output "substrate_apigateway_authenticator_role_arn" {
  value = module.substrate-apigateway-authenticator.role_arn
}

output "substrate_apigateway_authorizer_role_arn" {
  value = module.substrate-apigateway-authorizer.role_arn
}

output "validation_fqdn" {
  value = aws_route53_record.validation.fqdn
}
