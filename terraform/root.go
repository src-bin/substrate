package terraform

import (
	"os"
	"path"
	"text/template"
)

//go:generate go run ../tools/template/main.go -name gitignoreTemplate gitignore.template
//go:generate go run ../tools/template/main.go -name makefileTemplate Makefile.template

// Root writes a simple Makefile and .gitignore into the given directory,
// expecting the directory to be used as a root Terraform module.  The
// Makefile is essentially to make it easy to invoke Terraform from other
// directories in a single command and without involving a shell as an
// intermediary.  The .gitignore keeps Terraform plugin binaries and Lambda
// function zip files out of version control.
func Root(dirname string) error {
	if err := gitignore(dirname); err != nil {
		return err
	}
	if err := makefile(dirname); err != nil {
		return err
	}
	return nil
}

func gitignore(dirname string) error {
	f, err := os.Create(path.Join(dirname, ".gitignore"))
	if err != nil {
		return err
	}
	defer f.Close()
	tmpl, err := template.New(".gitignore").Parse(gitignoreTemplate())
	if err != nil {
		return err
	}
	return tmpl.Execute(f, nil)
}

func makefile(dirname string) error {
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
