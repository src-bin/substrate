package terraform

type Subnet struct {
	AvailabilityZone         Value
	CidrBlock, IPv6CidrBlock Value
	MapPublicIPOnLaunch      bool
	Provider                 ProviderAlias
	Tags                     Tags
	VpcId                    Value
}

func (s Subnet) Label() Value {
	if s.Tags.Name != "" {
		return Q(s.Tags.Name)
	}
	publicPrivate := "private"
	if s.MapPublicIPOnLaunch {
		publicPrivate = "public"
	}
	if s.Tags.Environment != "" && s.Tags.Quality != "" {
		return Qf("%s-%s-%s-%s", s.Tags.Environment, s.Tags.Quality, s.AvailabilityZone, publicPrivate)
	} else if s.Tags.Special != "" {
		return Qf("%s-%s-%s", s.Tags.Special, s.AvailabilityZone, publicPrivate)
	}
	return Q("")
}

func (Subnet) Template() string {
	return `resource "aws_subnet" {{.Label.Value}} {
	assign_ipv6_address_on_creation = true
	availability_zone = {{.AvailabilityZone.Value}}
	cidr_block = {{.CidrBlock.Value}}
	ipv6_cidr_block = {{.IPv6CidrBlock.Value}}
	map_public_ip_on_launch = {{.MapPublicIPOnLaunch}}
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
	vpc_id = {{.VpcId}}
}`
}
