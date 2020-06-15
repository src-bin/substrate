data "aws_iam_role" "admin" {
  name = "Administrator"
}

# Hoisted out of ../../lambda-function/global to allow logging while still
# running the Credential Factory directly as the Administrator role.
resource "aws_iam_role_policy_attachment" "cloudwatch" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
  role       = data.aws_iam_role.admin.name
}