package terraform

type EIP struct {
	Commented          bool // set by a command-line flag to control costs incurred by NAT Gateways
	InternetGatewayRef Value
	Label              Value
	Provider           ProviderAlias
	Tags               Tags
}

func (eip EIP) Ref() Value {
	return Uf("aws_eip.%s", eip.Label)
}

func (EIP) Template() string {
	return `{{if .Commented -}}
/* commented because substrate.nat-gateways contains "no"
{{end -}}
resource "aws_eip" {{.Label.Value}} {
  depends_on = [{{.InternetGatewayRef}}]
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
  tags = {{.Tags.Value}}
  vpc = true
}
{{- if .Commented}}
*/
{{- end}}`
}
