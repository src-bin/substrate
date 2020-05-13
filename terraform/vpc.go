package terraform

import (
	"fmt"
	"strings"
)

type VPC struct {
	CidrBlock Value
	Provider  ProviderAlias
	Tags      Tags
}

func (vpc VPC) CidrsubnetIPv4(newbits, netnum int) Value {
	return cidrsubnet(fmt.Sprintf("aws_vpc.%s.cidr_block", vpc.Label()), newbits, netnum)
}

func (vpc VPC) CidrsubnetIPv6(newbits, netnum int) Value {
	return cidrsubnet(fmt.Sprintf("aws_vpc.%s.ipv6_cidr_block", vpc.Label()), newbits, netnum)
}

func (vpc VPC) Label() Value {
	if vpc.Tags.Name != "" {
		return Q(vpc.Tags.Name)
	} else if vpc.Tags.Environment != "" && vpc.Tags.Quality != "" {
		return Qf("%s-%s", vpc.Tags.Environment, vpc.Tags.Quality)
	} else if vpc.Tags.Special != "" {
		return Q(vpc.Tags.Special)
	}
	return Q("")
}

func (vpc VPC) Ref() Value {
	return Uf("aws_vpc.%s.id", vpc.Label())
}

func (VPC) Template() string {
	return `resource "aws_vpc" {{.Label.Value}} {
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
		"Name" = {{.Label.Value}}
{{if .Tags.Quality -}}
		"Quality" = "{{.Tags.Quality}}"
{{end -}}
		"SubstrateVersion" = "{{.Tags.SubstrateVersion}}"
	}
}`
}

func cidrsubnet(prefix string, newbits, netnum int) Value {
	if !strings.Contains(prefix, "aws_vpc.") {
		prefix = fmt.Sprintf("%q", prefix)
	}
	return Uf("cidrsubnet(%s, %d, %d)", prefix, newbits, netnum)
}
