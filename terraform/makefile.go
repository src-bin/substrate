package terraform

import (
	"os"
	"path"
	"text/template"
)

func Makefile(dirname string) error {
	f, err := os.Create(path.Join(dirname, "Makefile"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New("Makefile").Parse(makefileTemplate())
	if err != nil {
		return err
	}
	return tmpl.Execute(f, nil)
}

func makefileTemplate() string {
	return `all:

apply: init
	terraform apply -auto-approve # XXX

init: .terraform

plan:
	terraform plan

.terraform:
	terraform init

.PHONY: all apply init plan
`
}
