package terraform

type ResourceShare struct {
	Provider ProviderAlias
	Tags     Tags
}

func (rs ResourceShare) Label() Value {
	if rs.Tags.Name != "" {
		return Q(rs.Tags.Name)
	}
	if rs.Tags.Environment != "" && rs.Tags.Quality != "" {
		return Qf("%s-%s", rs.Tags.Environment, rs.Tags.Quality)
	} else if rs.Tags.Special != "" {
		return Q(rs.Tags.Special)
	}
	return Q("")
}

func (rs ResourceShare) Ref() Value {
	return Uf("aws_ram_resource_share.%s.arn", rs.Label())
}

func (ResourceShare) Template() string {
	return `resource "aws_ram_resource_share" {{.Label.Value}} {
	allow_external_principals = false
	name = {{.Label.Value}}
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
