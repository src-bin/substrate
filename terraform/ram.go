package terraform

type PrincipalAssociation struct {
	Label                       Value
	Provider                    ProviderAlias
	Principal, ResourceShareArn Value
}

func (pa PrincipalAssociation) Ref() Value {
	return Uf("aws_resource_association.%s", pa.Label)
}

func (PrincipalAssociation) Template() string {
	return `resource "aws_ram_principal_association" {{.Label.Value}} {
  principal          = {{.Principal.Value}}
  provider           = {{.Provider}}
  resource_share_arn = {{.ResourceShareArn.Value}}
}`
}

type ResourceAssociation struct {
	Label                         Value
	Provider                      ProviderAlias
	ResourceArn, ResourceShareArn Value
}

func (ra ResourceAssociation) Ref() Value {
	return Uf("aws_resource_association.%s", ra.Label)
}

func (ResourceAssociation) Template() string {
	return `resource "aws_ram_resource_association" {{.Label.Value}} {
  provider           = {{.Provider}}
  resource_arn       = {{.ResourceArn.Value}}
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
  name                      = {{.Label.Value}}
  provider                  = {{.Provider}}
  tags                      = {{.Tags.Value}}
}`
}
