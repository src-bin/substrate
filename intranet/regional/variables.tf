variable "apigateway_role_arn" {
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
  type = string
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
