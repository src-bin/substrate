package terraform

type ResourceAssociation struct {
	Label                   Value
	Provider                ProviderAlias
	Resource, ResourceShare Value
}

func (ra ResourceAssociation) Ref() Value {
	return Uf("aws_ram_resource_share.%s.arn", ra.Label)
}

func (ResourceAssociation) Template() string {
	return `resource "aws_ram_resource_association" {{.Label.Value}} {
	provider = {{.Provider}}
	resource_arn = {{.Resource.Value}}
	resource_share_arn = {{.ResourceShare.Value}}
}`
}
