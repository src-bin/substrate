variable "apigateway_execution_arn" {
  type = string
}

variable "apigateway_role_arn" {
  type = string
}

variable "authorizer_id" {
  type = string
}

variable "lambda_role_arn" {
  type = string
}

variable "methods" {
  default = ["GET", "POST"] # only these browser-implemented methods are ever supported
  type    = list(string)
}

variable "parent_resource_id" {
  type = string
}

variable "proxy_destination_url" {
  type = string
}

variable "proxy_path_prefix" { # cannot contain '/' characters
  type = string
}

variable "rest_api_id" {
  type = string
}

variable "strip_path_prefix" {
  default = false
  type    = bool
}
