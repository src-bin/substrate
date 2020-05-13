package terraform

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
