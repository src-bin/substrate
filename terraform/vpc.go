package terraform

import (
	"fmt"
	"strings"
)

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
	provider = {{.Provider}}
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
	provider = {{.Provider}}
	tags = {{.Tags.Value}}
}`
}

func cidrsubnet(prefix string, newbits, netnum int) Value {
	if !strings.Contains(prefix, "aws_vpc.") {
		prefix = fmt.Sprintf("%q", prefix)
	}
	return Uf("cidrsubnet(%s, %d, %d)", prefix, newbits, netnum)
}
