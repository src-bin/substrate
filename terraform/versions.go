package terraform

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/src-bin/substrate/fileutil"
)

var TerraformVersion = "" // replaced at build time with the contents of terraform-version.txt; see Makefile

const (
	awsVersionConstraint      = "~> 3.49"
	externalVersionConstraint = "~> 2.1"
)

//go:generate go run ../tools/template/main.go -name versionsTemplate versions.tf.template

func versions(dirname string, configurationAliases []ProviderAlias) error {
	pathname := filepath.Join(dirname, "versions.tf")
	b, err := fileutil.ReadFile(pathname)

	if errors.Is(err, fs.ErrNotExist) {
		f, err := os.Create(pathname)
		if err != nil {
			return err
		}
		defer f.Close()
		tmpl, err := template.New("versions.tf").Parse(versionsTemplate())
		if err != nil {
			return err
		}
		return tmpl.Execute(f, struct {
			AWSVersionConstraint, ExternalVersionConstraint string
			ConfigurationAliases                            []ProviderAlias
			TerraformVersion                                string
		}{
			awsVersionConstraint, externalVersionConstraint,
			configurationAliases,
			TerraformVersion,
		})
	}

	// Use crude but at least precise regular expressions to upgrade provider
	// versions as necessary without disturbing additional providers that
	// folks may add.

	b = regexp.MustCompile(
		`# managed by Substrate; do not edit by hand`,
	).ReplaceAllLiteral(b, []byte(`# partially managed by Substrate; do not edit the archive, aws, or external providers by hand`))

	b = regexp.MustCompile( // remove in 2021.12
		`\s*archive\s*=\s*\{
\s*source\s*=\s*"hashicorp/archive"
\s*version\s*=\s*"(=|>=|~>)?\s*\d+(\.\d+)*"
\s*\}`,
	).ReplaceAllLiteral(b, []byte{})

	b = regexp.MustCompile(
		`source\s*=\s*"hashicorp/aws"
\s*version\s*=\s*"(=|>=|~>)?\s*\d+(\.\d+)*"`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`source = "hashicorp/aws"
      version = "%s"`,
		awsVersionConstraint,
	)))
	// TODO need to handle configuration_aliases for completeness (one customer was actually missing configuration_aliases because of this, though the consequences were extremely mild)

	b = regexp.MustCompile(
		`source\s*=\s*"hashicorp/external"
\s*version\s*=\s*"(=|>=|~>)?\s*\d+(\.\d+)*"`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`source = "hashicorp/external"
      version = "%s"`,
		externalVersionConstraint,
	)))

	b = regexp.MustCompile(
		`required_version\s*=\s*">?(= )?\d+\.\d+\.\d+"`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`required_version = "= %s"`,
		TerraformVersion,
	)))

	return ioutil.WriteFile(pathname, b, 0666)
}
