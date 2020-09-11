package lambdautil

import (
	"html/template"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
)

func RenderHTML(html string, v interface{}) (string, error) {
	tmpl, err := template.New("HTML").Funcs(template.FuncMap{
		"RegionFromAZ": func(az *string) string { return (*az)[:len(*az)-1] },
		"StringValue":  func(s *string) string { return aws.StringValue(s) },
	}).Parse(html)
	if err != nil {
		return "", err
	}
	builder := &strings.Builder{}
	if err = tmpl.Execute(builder, v); err != nil {
		return "", err
	}
	return builder.String(), nil
}
