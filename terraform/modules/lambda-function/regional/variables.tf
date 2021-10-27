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

variable "progname" {
  default = ""
  type    = string
}

variable "role_arn" {
  type = string
}
