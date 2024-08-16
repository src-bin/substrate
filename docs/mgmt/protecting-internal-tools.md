# Protecting internal tools

The tools Substrate manages under your Intranet (the Accounts page that facilitates logging into the AWS Console, the Credential Factory, and the Instance Factory) are probably not the only internal tools you're going to operate as a part of your business and your Intranet can protect your other internal tools with the same SSO, HTTP Strict Transport Security, separate cookie scope, robust serverless implementation, and regional fault-tolerance.

In this example, we're going to route Intranet requests for `/example` to a Lambda function. (The details of the Lambda function refered to here as `aws_lambda_function.example` are left as an exercise to the reader.) Add Terraform code like the following to `modules/intranet/regional/example.tf`:

    data "aws_apigatewayv2_apis" "substrate" {
      name          = "Substrate"
      protocol_type = "HTTP"
    }

    data "aws_apigatewayv2_api" "substrate" {
      api_id = tolist(data.aws_apigatewayv2_apis.substrate.ids)[0]
    }

    resource "aws_apigatewayv2_integration" "example" {
      api_id             = data.aws_apigatewayv2_api.substrate.id
      integration_method = "POST"
      integration_type   = "AWS_PROXY" # or "HTTP_PROXY" with connection_id, connection_type = "VPC_LINK" and other attributes
      integration_uri    = aws_lambda_function.example.invoke_arn
    }

    resource "aws_apigatewayv2_route" "example" {
      api_id             = data.aws_apigatewayv2_api.substrate.id
      authorization_type = "CUSTOM"
      authorizer_id      = data.aws_apigatewayv2_api.substrate.tags["AuthorizerId"]
      route_key          = "ANY /example" # or "ANY /example/{proxy+}" or a more specific HTTP method
      target             = "integrations/${aws_apigatewayv2_integration.example.id}"
    }

    resource "aws_lambda_permission" "example" {
      action        = "lambda:InvokeFunction"
      function_name = aws_lambda_function.example.function_name
      principal     = "apigateway.amazonaws.com"
      source_arn    = "${data.aws_apigatewayv2_api.substrate.execution_arn}/*"
    }

The `aws_apigatewayv2_integration` does not have to have `integration_type = "AWS_PROXY"`. Beware, though, that setting `integration_type = "HTTP_PROXY"` without also configuring VPC link with `connection_type = "VPC_LINK"`, a `connection_id` attribute, and an `aws_apigatewayv2_vpc_link` resource is almost certainly a security vulnerability.

Note, too, that you do not have to use Terraform to route requests from your Intranet to your internal tools. Substrate may provide facilities to natively manage these internal tool integrations in the future.
