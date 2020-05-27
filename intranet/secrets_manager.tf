data "aws_iam_policy_document" "client-secret" {
  statement {
    actions = ["secretsmanager:GetSecretValue"]
    principals {
      identifiers = [
        module.okta-authenticator.role_arn,
        module.okta-authorizer.role_arn,
      ]
      type = "AWS"
    }
    resources = ["*"]
  }
}

resource "aws_secretsmanager_secret" "client-secret" {
  name                    = "OktaClientSecret-${var.okta_client_id}"
  policy                  = data.aws_iam_policy_document.client-secret.json
  recovery_window_in_days = 0 # if this is set to anything but zero and the policy is invalid, things get weird
  tags = {
    "Manager" = "Terraform"
  }
}
