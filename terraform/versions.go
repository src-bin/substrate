package terraform

import (
	"os"
	"path/filepath"
	"text/template"
)

//go:generate go run ../tools/template/main.go -name versionsTemplate versions.tf

const RequiredVersion = "0.13.6"

func versions(dirname string) error {
	f, err := os.Create(filepath.Join(dirname, "versions.tf"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("versions.tf").Parse(versionsTemplate())
	if err != nil {
		return err
	}
	return tmpl.Execute(f, struct{ RequiredVersion string }{RequiredVersion})
}
