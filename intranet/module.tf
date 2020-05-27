variable "okta_client_id" {}

variable "okta_client_secret_timestamp" {} # TODO it's awkward to have to apply this Terraform in order to know how to set this

variable "okta_hostname" {}

variable "stage_name" {}

output "url" {
  value = "https://${aws_api_gateway_rest_api.intranet.id}.execute_api.${data.aws_region.current.name}.amazonaws.com/${var.stage_name}"
}
