package terraform

// managed by go generate; do not edit by hand

func lambdaFunctionGlobalTemplate() map[string]string {
	return map[string]string{
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
		"variables.tf": `variable "name" {}

variable "policy" {}
`,
		"outputs.tf":   `output "role_arn" {
  value = aws_iam_role.role.arn
}
`,
	}
}
