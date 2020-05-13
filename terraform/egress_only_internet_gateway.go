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
