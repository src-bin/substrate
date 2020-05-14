package terraform

type EgressOnlyInternetGateway struct {
	Label    Value
	Provider ProviderAlias
	Tags     Tags
	VpcId    Value
}

func (egw EgressOnlyInternetGateway) Ref() Value {
	return Uf("aws_egress_only_internet_gateway.%s", egw.Label)
}

func (EgressOnlyInternetGateway) Template() string {
	return `resource "aws_egress_only_internet_gateway" {{.Label.Value}} {
	provider = {{.Provider}}
	tags = {
{{- if .Tags.Environment}}
		"Environment" = "{{.Tags.Environment}}"
{{- end}}
		"Manager" = "{{.Tags.Manager}}"
{{- if .Tags.Name}}
		"Name" = "{{.Tags.Name}}"
{{- end}}
{{- if .Tags.Quality}}
		"Quality" = "{{.Tags.Quality}}"
{{- end}}
		"SubstrateVersion" = "{{.Tags.SubstrateVersion}}"
	}
	vpc_id = {{.VpcId.Value}}
}`
}

type InternetGateway struct {
	Label    Value
	Provider ProviderAlias
	Tags     Tags
	VpcId    Value
}

func (igw InternetGateway) Ref() Value {
	return Uf("aws_internet_gateway.%s", igw.Label)
}

func (InternetGateway) Template() string {
	return `resource "aws_internet_gateway" {{.Label.Value}} {
	provider = {{.Provider}}
	tags = {
{{- if .Tags.Environment}}
		"Environment" = "{{.Tags.Environment}}"
{{- end}}
		"Manager" = "{{.Tags.Manager}}"
{{- if .Tags.Name}}
		"Name" = "{{.Tags.Name}}"
{{- end}}
{{- if .Tags.Quality}}
		"Quality" = "{{.Tags.Quality}}"
{{- end}}
		"SubstrateVersion" = "{{.Tags.SubstrateVersion}}"
	}
	vpc_id = {{.VpcId.Value}}
}`
}

type NATGateway struct {
	InternetGatewayRef Value
	Label              Value
	Provider           ProviderAlias
	SubnetId           Value
	Tags               Tags
}

func (ngw NATGateway) Ref() Value {
	return Uf("aws_nat_gateway.%s", ngw.Label)
}

func (NATGateway) Template() string {
	return `resource "aws_nat_gateway" {{.Label.Value}} {
	allocation_id = aws_eip.{{.Label}}.id
	provider = {{.Provider}}
	subnet_id = {{.SubnetId.Value}}
	tags = {{.Tags.Value}}
}`
}
