variable "apigateway_execution_arn" {}

variable "filename" {}

variable "name" {}

variable "policy" {}

output "function_arn" {
  value = aws_lambda_function.function.arn
}

output "invoke_arn" {
  value = aws_lambda_function.function.invoke_arn
}

output "role_arn" {
  value = aws_iam_role.role.arn
}
