package terraform

type ResourceAssociation struct {
	Label                         Value
	Provider                      ProviderAlias
	ResourceArn, ResourceShareArn Value
}

func (ResourceAssociation) Template() string {
	return `resource "aws_ram_resource_association" {{.Label.Value}} {
	provider = {{.Provider}}
	resource_arn = {{.ResourceArn.Value}}
	resource_share_arn = {{.ResourceShareArn.Value}}
}`
}

type ResourceShare struct {
	Label    Value
	Provider ProviderAlias
	Tags     Tags
}

func (rs ResourceShare) Ref() Value {
	return Uf("aws_ram_resource_share.%s", rs.Label)
}

func (ResourceShare) Template() string {
	return `resource "aws_ram_resource_share" {{.Label.Value}} {
	allow_external_principals = false
	name = {{.Label.Value}}
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
}`
}
