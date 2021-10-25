variable "authorizer_id" {
  type = string
}

variable "destination" {
  type = string
}

variable "invoke_arn" {
  type = string
}

variable "methods" {
  default = ["GET", "POST"] # only these browser-implemented methods are ever supported
  type    = list(string)
}

variable "parent_resource_id" {
  type = string
}

variable "path_part" {
  type = string
}

variable "rest_api_id" {
  type = string
}

variable "role_arn" {
  type = string
}

variable "strip_prefix" {
  default = false
  type    = bool
}
