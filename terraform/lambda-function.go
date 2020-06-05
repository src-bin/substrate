package terraform

// managed by go generate; do not edit by hand

func lambdaFunctionTemplate() map[string]string {
	return map[string]string{
		"module.tf":    `variable "apigateway_execution_arn" {}

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
`,
		"cloudwatch.tf":`resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.name}"
  retention_in_days = 1
  tags = {
    Manager = "Terraform"
  }
}
`,
		"lambda.tf":    `resource "aws_lambda_function" "function" {
  filename         = var.filename
  function_name    = var.name
  handler          = var.name
  memory_size      = 128 # default
  role             = aws_iam_role.role.arn
  runtime          = "go1.x"
  source_code_hash = filebase64sha256(var.filename)
  tags = {
    Manager = "Terraform"
    Name    = var.name
  }
  timeout = 60
}

resource "aws_lambda_permission" "permission" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.function.function_name
  principal     = "apigateway.amazonaws.com"
  #source_arn    = var.apigateway_execution_arn
}
`,
		"iam.tf":       `data "aws_iam_policy_document" "lambda-trust" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["lambda.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_policy" "policy" {
  name   = var.name
  policy = var.policy
}

resource "aws_iam_role" "role" {
  assume_role_policy = data.aws_iam_policy_document.lambda-trust.json
  name               = var.name
}

resource "aws_iam_role_policy_attachment" "policy" {
  policy_arn = aws_iam_policy.policy.arn
  role       = aws_iam_role.role.name
}

resource "aws_iam_role_policy_attachment" "cloudwatch" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
  role       = aws_iam_role.role.name
}
`,
	}
}
