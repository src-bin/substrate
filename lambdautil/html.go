package lambdautil

import (
	"html/template"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
)

//go:generate go run ../tools/template/main.go -name navTemplate nav.html

func RenderHTML(html string, v interface{}) (string, error) {
	tmpl, err := template.Must(template.New("nav").Parse(navTemplate())).New("HTML").Funcs(template.FuncMap{
		"RegionFromAZ": func(az *string) string { return (*az)[:len(*az)-1] },
		"ToString":     func(s *string) string { return aws.ToString(s) },
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
