package terraform

import (
	"fmt"
)

type VPC struct {
	Label     string // defaults to Name()
	CidrBlock string
	Provider  ProviderAlias
	Tags      Tags
}

func (vpc VPC) Name() string {
	if vpc.Tags.Environment != "" && vpc.Tags.Quality != "" {
		return fmt.Sprintf("%s-%s", vpc.Tags.Environment, vpc.Tags.Quality)
	} else if vpc.Tags.Special != "" {
		return vpc.Tags.Special
	}
	return ""
}

func (VPC) Template() string {
	return `resource "aws_vpc" "{{if .Label}}{{.Label}}{{else}}{{.Name}}{{end}}" {
	assign_generated_ipv6_cidr_block = true
	cidr_block = "{{.CidrBlock}}"
	enable_dns_hostnames = true
	enable_dns_support = true
	provider = {{.Provider}}
	tags = {
{{if .Tags.Environment -}}
		"Environment" = "{{.Tags.Environment}}"
{{end -}}
		"Manager" = "{{.Tags.Manager}}"
{{if .Name -}}
		"Name" = "{{.Name}}"
{{end -}}
{{if .Tags.Quality -}}
		"Quality" = "{{.Tags.Quality}}"
{{end -}}
		"SubstrateVersion" = "{{.Tags.SubstrateVersion}}"
	}
}`
}
