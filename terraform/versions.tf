# managed by Substrate; do not edit by hand

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= {{.RequiredProvidersAWSVersion}}"
    }
  }
  required_version = "= {{.RequiredVersion}}"
}
