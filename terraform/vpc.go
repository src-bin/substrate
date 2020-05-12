package terraform

import (
	"fmt"
	"strings"
)

type VPC struct {
	Label     Value // defaults to Name()
	CidrBlock Value
	Provider  ProviderAlias
	Tags      Tags
}

func (vpc VPC) CidrsubnetIPv4(newbits, netnum int) Value {
	return cidrsubnet(fmt.Sprintf("aws_vpc.%s.cidr_block", vpc.Name()), newbits, netnum)
}

func (vpc VPC) CidrsubnetIPv6(newbits, netnum int) Value {
	return cidrsubnet(fmt.Sprintf("aws_vpc.%s.ipv6_cidr_block", vpc.Name()), newbits, netnum)
}

func (vpc VPC) Name() Value {
	if vpc.Label != nil && !vpc.Label.Empty() {
		return vpc.Label
	} else if vpc.Tags.Environment != "" && vpc.Tags.Quality != "" {
		return Qf("%s-%s", vpc.Tags.Environment, vpc.Tags.Quality)
	} else if vpc.Tags.Special != "" {
		return Q(vpc.Tags.Special)
	}
	return Q("")
}

func (VPC) Template() string {
	return `resource "aws_vpc" {{.Name.Value}} {
	assign_generated_ipv6_cidr_block = true
	cidr_block = {{.CidrBlock.Value}}
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

func (vpc VPC) label() Value {
	return vpc.Name()
}

func cidrsubnet(prefix string, newbits, netnum int) Value {
	if !strings.Contains(prefix, "aws_vpc.") {
		prefix = fmt.Sprintf("%q", prefix)
	}
	return Uf("cidrsubnet(%s, %d, %d)", prefix, newbits, netnum)
}
