package terraform

type EIP struct {
	InternetGatewayRef Value
	Label              Value
	Provider           ProviderAlias
	Tags               Tags
}

func (eip EIP) Ref() Value {
	return Uf("aws_eip.%s", eip.Label)
}

func (EIP) Template() string {
	return `resource "aws_eip" {{.Label.Value}} {
  depends_on = [{{.InternetGatewayRef}}]
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  tags = {{.Tags.Value}}
  vpc = true
}`
}
