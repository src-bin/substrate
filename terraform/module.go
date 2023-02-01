package terraform

type Module struct {
	Arguments map[string]Value
	DependsOn ValueSlice
	Label     Value
	Provider  ProviderAlias
	Providers map[ProviderAlias]ProviderAlias
	Source    Value
}

func (m Module) Ref() Value {
	return Uf("module.%s", m.Label)
}

func (Module) Template() string {
	return `module {{.Label.Value}} {
{{- range $k, $v := .Arguments }}
  {{$k}} = {{$v.Value}}
{{- end}}

{{- if .DependsOn}}
    depends_on = {{.DependsOn.Value}}
{{- end}}
{{- if or .Provider .Providers}}
  providers = {
{{- if .Provider}}
    aws = {{.Provider}}
{{- end}}
{{- range $k, $v := .Providers }}
  {{$k}} = {{$v}}
{{- end}}
  }
{{- end}}
  source = {{.Source.Value}}
}`
}
