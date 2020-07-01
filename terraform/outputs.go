package terraform

type Output struct {
	Label Value
	Value Value
}

func (o Output) Ref() Value {
	panic("Output.Ref doesn't make sense")
}

func (Output) Template() string {
	return `output {{.Label.Value}} {
  value = {{.Value.Value}}
}`
}
