variable "apigateway_role_arn" {}

variable "okta_client_id" {}

variable "okta_client_secret_timestamp" {} # TODO it's awkward to have to apply this Terraform in order to know how to set this

variable "okta_hostname" {}

variable "stage_name" {}

variable "substrate_instance_factory_role_arn" {}

variable "substrate_okta_authenticator_role_arn" {}

variable "substrate_okta_authorizer_role_arn" {}
