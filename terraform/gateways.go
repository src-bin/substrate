package terraform

type EgressOnlyInternetGateway struct {
	Provider ProviderAlias
	Tags     Tags
	VpcId    Value
}

func (egw EgressOnlyInternetGateway) Label() Value {
	return Q(egw.Tags.Name)
}

func (egw EgressOnlyInternetGateway) Ref() Value {
	return Uf("aws_internet_gateway.%s.id", egw.Label())
}

func (EgressOnlyInternetGateway) Template() string {
	return `resource "aws_egress_only_internet_gateway" {{.Label.Value}} {
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
	vpc_id = {{.VpcId.Value}}
}`
}

type InternetGateway struct {
	Provider ProviderAlias
	Tags     Tags
	VpcId    Value
}

func (igw InternetGateway) Label() Value {
	return Q(igw.Tags.Name)
}

func (igw InternetGateway) Ref() Value {
	return Uf("aws_internet_gateway.%s.id", igw.Label())
}

func (InternetGateway) Template() string {
	return `resource "aws_internet_gateway" {{.Label.Value}} {
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
	vpc_id = {{.VpcId.Value}}
}`
}

type NATGateway struct {
	Provider ProviderAlias
	SubnetId Value
	Tags     Tags
}

func (ngw NATGateway) Label() Value {
	if ngw.Tags.Name != "" {
		return Q(ngw.Tags.Name)
	}
	if ngw.Tags.Environment != "" && ngw.Tags.Quality != "" {
		return Qf("%s-%s-%s", ngw.Tags.Environment, ngw.Tags.Quality, ngw.Tags.AvailabilityZone)
	} else if ngw.Tags.Special != "" {
		return Qf("%s-%s", ngw.Tags.Special, ngw.Tags.AvailabilityZone)
	}
	return Q("")
}

func (ngw NATGateway) Ref() Value {
	return Uf("aws_nat_gateway.%s.id", ngw.Label())
}

func (NATGateway) Template() string {
	return `"aws_eip" {{.Label.Value}} {
	provider = {{.Provider}}
}

resource "aws_nat_gateway" {{.Label.Value}} {
	provider = {{.Provider}}
	subnet_id = {{.SubnetId.Value}}
	tags = {
		"AvailabilityZone" = "{{.Tags.AvailabilityZone}}"
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
