package terraform

type TimeSleep struct {
	CreateDuration Value
	DependsOn      ValueSlice
	ForEach        Value
	Label          Value
	Provider       ProviderAlias
}

func (ts TimeSleep) Ref() Value {
	return Uf("time_sleep.%s", ts.Label)
}

func (TimeSleep) Template() string {
	return `resource "time_sleep" {{.Label.Value}} {
{{- if .CreateDuration}}
  create_duration = {{.CreateDuration.Value}}
{{- end}}
{{- if .DependsOn}}
  depends_on = {{.DependsOn.Value}}
{{- end}}
{{- if .ForEach}}
  for_each = {{.ForEach.Value}}
{{- end}}
{{- if .Provider}}
  provider = {{.Provider}}
{{- end}}
}
`
}
