variable "apigateway_role_arn" {
  type = string
}

variable "okta_client_id" {
  type = string
}

variable "okta_client_secret_timestamp" {
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

variable "substrate_instance_factory_role_arn" {
  type = string
}

variable "substrate_okta_authenticator_role_arn" {
  type = string
}

variable "substrate_okta_authorizer_role_arn" {
  type = string
}
