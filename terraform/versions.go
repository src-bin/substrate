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

var TerraformVersion = "" // replaced at build time with the contents of terraform.version; see Makefile

const (
	awsVersionConstraint      = "~> 5.14"
	externalVersionConstraint = "~> 2.1"
)

//go:generate go run ../tools/template/main.go -name versionsTemplate versions.tf.template

func versions(dirname string, configurationAliases []ProviderAlias, versionConstraints bool) error {
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
			VersionConstraints                              bool
		}{
			awsVersionConstraint, externalVersionConstraint,
			configurationAliases,
			TerraformVersion,
			versionConstraints,
		})
	}

	// Use crude but at least precise regular expressions to upgrade provider
	// versions as necessary without disturbing additional providers that
	// folks may add.
	var replacement string

	b = regexp.MustCompile(
		`# managed by Substrate; do not edit by hand`,
	).ReplaceAllLiteral(b, []byte(
		`# partially managed by Substrate; do not edit the aws or external providers by hand`,
	))

	replacement = "source = \"hashicorp/aws\"\n"
	if versionConstraints {
		replacement += fmt.Sprintf("      version = \"%s\"\n", awsVersionConstraint)
	}
	b = regexp.MustCompile(
		`source\s*=\s*"hashicorp/aws"
(\s*version\s*=\s*"\s*(=|>=|~>)?\s*\d+(\.\d+)*"
)?`,
	).ReplaceAllLiteral(b, []byte(replacement))
	// TODO need to handle configuration_aliases for completeness (one customer was actually missing configuration_aliases because of this, though the consequences were extremely mild)

	replacement = "source = \"hashicorp/external\"\n"
	if versionConstraints {
		replacement += fmt.Sprintf("      version = \"%s\"\n", externalVersionConstraint)
	}
	b = regexp.MustCompile(
		`source\s*=\s*"hashicorp/external"
(\s*version\s*=\s*"\s*(=|>=|~>)?\s*\d+(\.\d+)*"
)?`,
	).ReplaceAllLiteral(b, []byte(replacement))

	replacement = "" // since this doesn't leave anything to anchor us, this is a one-way door (for now)
	if versionConstraints {
		replacement += fmt.Sprintf("required_version = \"= %s\"\n", TerraformVersion)
	}
	b = regexp.MustCompile(
		`(  )?required_version\s*=\s*"\s*>?(= )?\d+\.\d+\.\d+"
`, // if later we need to make this reversible, look for the trailing }\n}\n$
	).ReplaceAllLiteral(b, []byte(replacement))

	if err := ioutil.WriteFile(pathname, b, 0666); err != nil {
		return err
	}
	return Fmt(dirname)
}
