package terraform

// managed by go generate; do not edit by hand

func versionsTemplate() string {
	return `# managed by Substrate; do not edit by hand

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
  required_version = "= {{.RequiredVersion}}"
}
`
}
