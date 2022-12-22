variable "apigateway_execution_arn" {
  type = string
}

variable "environment_variables" {
  default = {}
  type    = map(string)
}

variable "filename" {
  type = string
}

variable "name" {
  type = string
}

variable "role_arn" {
  type = string
}

variable "security_group_ids" {
  default = []
  type    = list(string)
}

variable "source_code_hash" {
  type = string
}

variable "subnet_ids" {
  default = []
  type    = list(string)
}
