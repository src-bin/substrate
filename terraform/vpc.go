package terraform

import (
	"fmt"
	"strings"
)

type DataSubnet struct {
	ForEach, Id Value
	Label       Value
	Provider    ProviderAlias
}

func (d DataSubnet) Ref() Value {
	return Uf("data.aws_subnet.%s", d.Label)
}

func (DataSubnet) Template() string {
	return `data "aws_subnet" {{.Label.Value}} {
{{- if .ForEach}}
  for_each = {{.ForEach.Value}}
{{- end}}
  id = {{.Id.Value}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
}
`
}

type DataSubnetIds struct {
	Label    Value
	Provider ProviderAlias
	Tags     Tags
	VpcId    Value
}

func (d DataSubnetIds) Ref() Value {
	return Uf("data.aws_subnet_ids.%s", d.Label)
}

func (DataSubnetIds) Template() string {
	return `data "aws_subnet_ids" {{.Label.Value}} {
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
{{- if .Tags}}
  tags = {{.Tags.Value}}
{{- end}}
  vpc_id = {{.VpcId.Value}}
}
`
}

type DataVPC struct {
	Label    Value
	Provider ProviderAlias
	Tags     Tags
}

func (d DataVPC) Ref() Value {
	return Uf("data.aws_vpc.%s", d.Label)
}

func (DataVPC) Template() string {
	return `data "aws_vpc" {{.Label.Value}} {
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
{{- if .Tags}}
  tags = {{.Tags.Value}}
{{- end}}
}
`
}

type Subnet struct {
	AvailabilityZone         Value
	CidrBlock, IPv6CidrBlock Value
	Label                    Value
	MapPublicIPOnLaunch      bool
	Provider                 ProviderAlias
	Tags                     Tags
	VpcId                    Value
}

func (s Subnet) Ref() Value {
	return Uf("aws_subnet.%s", s.Label)
}

func (Subnet) Template() string {
	return `resource "aws_subnet" {{.Label.Value}} {
  assign_ipv6_address_on_creation = true
  availability_zone = {{.AvailabilityZone.Value}}
  cidr_block = {{.CidrBlock.Value}}
  ipv6_cidr_block = {{.IPv6CidrBlock.Value}}
  map_public_ip_on_launch = {{.MapPublicIPOnLaunch}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  tags = {{.Tags.Value}}
  vpc_id = {{.VpcId.Value}}
}`
}

type VPC struct {
	CidrBlock Value
	Label     Value
	Provider  ProviderAlias
	Tags      Tags
}

func (vpc VPC) CidrsubnetIPv4(newbits, netnum int) Value {
	return cidrsubnet(fmt.Sprintf("aws_vpc.%s.cidr_block", vpc.Label), newbits, netnum)
}

func (vpc VPC) CidrsubnetIPv6(newbits, netnum int) Value {
	return cidrsubnet(fmt.Sprintf("aws_vpc.%s.ipv6_cidr_block", vpc.Label), newbits, netnum)
}

func (vpc VPC) Ref() Value {
	return Uf("aws_vpc.%s", vpc.Label)
}

func (VPC) Template() string {
	return `resource "aws_vpc" {{.Label.Value}} {
  assign_generated_ipv6_cidr_block = true
  cidr_block = {{.CidrBlock.Value}}
  enable_dns_hostnames = true
  enable_dns_support = true
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  tags = {{.Tags.Value}}
}`
}

type VPCEndpoint struct {
	Label         Value
	Provider      ProviderAlias
	RouteTableIds ValueSlice
	ServiceName   Value
	Tags          Tags
	VpcId         Value
}

func (vpce VPCEndpoint) Ref() Value {
	return Uf("aws_vpc_endpoint.%s", vpce.Label)
}

func (VPCEndpoint) Template() string {
	return `resource "aws_vpc_endpoint" {{.Label.Value}} {
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
{{- if .RouteTableIds}}
  route_table_ids = {{.RouteTableIds.Value}}
{{- end}}
  service_name = {{.ServiceName.Value}}
  tags = {{.Tags.Value}}
  vpc_id = {{.VpcId.Value}}
}`
}

func cidrsubnet(prefix string, newbits, netnum int) Value {
	if !strings.Contains(prefix, "aws_vpc.") {
		prefix = fmt.Sprintf("%q", prefix)
	}
	return Uf("cidrsubnet(%s, %d, %d)", prefix, newbits, netnum)
}
