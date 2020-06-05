resource "aws_lambda_function" "function" {
  depends_on       = [null_resource.zip]
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

resource "null_resource" "zip" {
  provisioner "local-exec" {
    command = "touch -t 197001010000 $GOBIN/${var.name}"
  }
  provisioner "local-exec" {
    command = "zip -X -j ${var.filename} $GOBIN/${var.name}"
  }
  triggers = { timestamp = timestamp() } # trigger every time
}
