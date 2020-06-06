output "apigateway_role_arn" {
  value = aws_iam_role.apigateway.arn
}

output "substrate_instance_factory_role_arn" {
  value = module.substrate-instance-factory.role_arn
}

output "substrate_okta_authenticator_role_arn" {
  value = module.substrate-okta-authenticator.role_arn
}
output "substrate_okta_authorizer_role_arn" {
  value = module.substrate-okta-authorizer.role_arn
}
