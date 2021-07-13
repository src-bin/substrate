package terraform

// managed by go generate; do not edit by hand

func versionsTemplate() string {
	return `# partially managed by Substrate; do not edit the archive, aws, or external providers by hand

terraform {
  required_providers {
    archive = {
      source  = "hashicorp/archive"
      version = ">= {{.ArchiveVersion}}"
    }
    aws = {
{{- if .ConfigurationAliases}}
      configuration_aliases = [
{{- range .ConfigurationAliases }}
        {{.}},
{{- end}}
      ]
{{- end}}
      source  = "hashicorp/aws"
      version = ">= {{.AWSVersion}}"
    }
    external = {
      source  = "hashicorp/external"
      version = ">= {{.ExternalVersion}}"
    }
  }
  required_version = "= {{.TerraformVersion}}"
}
`
}
