package terraform

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

const (
	AWSProviderVersionConstraintFilename = "terraform-aws.version-constraint"

	DefaultAWSProviderVersionConstraint = "~> 5.35"

	RequiredVersionFilename = "terraform.version"

	externalProviderVersionConstraint = "~> 2.1"
)

var DefaultRequiredVersion = "" // replaced at build time with the contents of terraform.version; see Makefile

//go:generate go run ../tools/template/main.go -name versionsTemplate versions.tf.template

func AWSProviderVersionConstraint() string {
	b, err := os.ReadFile(AWSProviderVersionConstraintFilename)
	if errors.Is(err, fs.ErrNotExist) {
		return DefaultAWSProviderVersionConstraint
	}
	if err != nil {
		ui.Fatal(err)
	}
	return fileutil.Tidy(b)
}

func RequiredVersion() string {
	b, err := os.ReadFile(RequiredVersionFilename)
	if errors.Is(err, fs.ErrNotExist) {
		return DefaultRequiredVersion
	}
	if err != nil {
		ui.Fatal(err)
	}
	return fileutil.Tidy(b)
}

func versions(dirname string, configurationAliases []ProviderAlias, versionConstraints bool) error {
	pathname := filepath.Join(dirname, "versions.tf")
	b, err := os.ReadFile(pathname)

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
			AWSProviderVersionConstraint      string
			ExternalProviderVersionConstraint string
			ConfigurationAliases              []ProviderAlias
			RequiredVersion                   string
			VersionConstraints                bool
		}{
			AWSProviderVersionConstraint(),
			externalProviderVersionConstraint,
			configurationAliases,
			RequiredVersion(),
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
		replacement += fmt.Sprintf("      version = \"%s\"\n", AWSProviderVersionConstraint())
	}
	b = regexp.MustCompile(
		`source\s*=\s*"hashicorp/aws"
(\s*version\s*=\s*"\s*(=|>=|~>)?\s*\d+(\.\d+)*"
)?`,
	).ReplaceAllLiteral(b, []byte(replacement))
	// TODO need to handle configuration_aliases for completeness (one customer was actually missing configuration_aliases because of this, though the consequences were extremely mild)

	replacement = "source = \"hashicorp/external\"\n"
	if versionConstraints {
		replacement += fmt.Sprintf("      version = \"%s\"\n", externalProviderVersionConstraint)
	}
	b = regexp.MustCompile(
		`source\s*=\s*"hashicorp/external"
(\s*version\s*=\s*"\s*(=|>=|~>)?\s*\d+(\.\d+)*"
)?`,
	).ReplaceAllLiteral(b, []byte(replacement))

	replacement = "" // since this doesn't leave anything to anchor us, this is a one-way door (for now)
	if versionConstraints {
		replacement += fmt.Sprintf("required_version = \"= %s\"\n", RequiredVersion())
	}
	b = regexp.MustCompile(
		`(  )?required_version\s*=\s*"\s*>?(= )?\d+\.\d+\.\d+"
`, // if later we need to make this reversible, look for the trailing }\n}\n$
	).ReplaceAllLiteral(b, []byte(replacement))

	if err := os.WriteFile(pathname, b, 0666); err != nil {
		return err
	}
	return Fmt(dirname)
}
