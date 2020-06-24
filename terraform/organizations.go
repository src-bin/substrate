package terraform

type Organization struct {
	Label    Value
	Provider ProviderAlias
}

func (o Organization) Ref() Value {
	return Uf("data.aws_organizations_organization.%s", o.Label)
}

func (Organization) Template() string {
	return `data "aws_organizations_organization" {{.Label.Value}} {
  provider = {{.Provider}}
}`
}
