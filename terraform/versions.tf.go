package terraform

// managed by go generate; do not edit by hand

func versionsTemplate() string {
	return `# managed by Substrate; do not edit by hand

terraform {
  required_providers {
    archive = {
      source  = "hashicorp/archive"
      version = "= 2.0.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "= 3.26.0"
    }
    external = {
      source  = "hashicorp/external"
      version = "= 2.0.0"
    }
  }
  required_version = "= 0.15.5"
}
`
}
