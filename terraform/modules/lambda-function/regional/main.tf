resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/${var.name}"
  retention_in_days = 7
}

resource "aws_lambda_function" "function" {
  architectures = ["arm64"]
  depends_on    = [aws_cloudwatch_log_group.lambda]
  environment {
    variables = merge(
      { "PREVENT_EMPTY_ENVIRONMENT" = "lambda:CreateFunction fails when given an empty Environment" },
      var.environment_variables,
    )
  }
  filename         = var.filename
  function_name    = var.name
  handler          = "bootstrap"
  memory_size      = 128 # default
  role             = var.role_arn
  runtime          = "provided.al2"
  source_code_hash = var.source_code_hash
  tags = {
    Name = var.name
  }
  timeout = 29
  vpc_config {
    security_group_ids = var.security_group_ids
    subnet_ids         = var.subnet_ids
  }
}

resource "aws_lambda_permission" "permission" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.function.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = var.apigateway_execution_arn
}
