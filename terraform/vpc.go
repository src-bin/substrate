package terraform

import "fmt"

type VPC struct {
	Label                         string
	CidrBlock                     string
	Environment, Quality, Special string
}

func (vpc VPC) Name() string {
	if vpc.Environment != "" && vpc.Quality != "" {
		return fmt.Sprintf("%s-%s", vpc.Environment, vpc.Quality)
	} else if vpc.Special != "" {
		return vpc.Special
	}
	return ""
}

func (VPC) Template() string {
	return `resource "aws_vpc" "{{.Label}}" {
	assign_generated_ipv6_cidr_block = true
	cidr_block = "{{.CidrBlock}}"
	enable_dns_hostnames = true
	enable_dns_support = true
	tags = {
{{if .Environment -}}
		"Environment" = "{{.Environment}}"
{{end -}}
		"Manager" = "Terraform"
{{if .Name -}}
		"Name" = "{{.Name}}"
{{end -}}
{{if .Quality -}}
		"Quality" = "{{.Quality}}"
{{end -}}
		"SubstrateVersion" = "TODO"
	}
}`
}
