package terraform

// managed by go generate; do not edit by hand

func versionsTemplate() string {
	return `# managed by Substrate; do not edit by hand

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= 3.26.0"
    }
  }
  required_version = "= 0.13.6"
}
`
}
