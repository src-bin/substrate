data "archive_file" "zip" {
  output_path = var.filename
  source_file = "${data.external.gobin.result.GOBIN}/${var.name}"
  type        = "zip"
}

data "external" "gobin" {
  program = ["/bin/sh", "-c", "echo \"{\\\"GOBIN\\\":\\\"$GOBIN\\\"}\""]
}

resource "aws_lambda_function" "function" {
  filename         = var.filename
  function_name    = var.name
  handler          = var.name
  memory_size      = 128 # default
  role             = var.role_arn
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
