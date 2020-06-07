package lambdautil

import (
	"html/template"
	"strings"
)

func RenderHTML(html string, v interface{}) (string, error) {
	tmpl, err := template.New("HTML").Parse(html)
	if err != nil {
		return "", err
	}
	builder := &strings.Builder{}
	if err = tmpl.Execute(builder, v); err != nil {
		return "", err
	}
	return builder.String(), nil
}
