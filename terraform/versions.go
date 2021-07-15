package terraform

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/src-bin/substrate/fileutil"
)

var TerraformVersion = "" // replaced at build time with the contents of terraform-version.txt; see Makefile

const (
	archiveVersion  = "2.2.0"
	awsVersion      = "3.49.0"
	externalVersion = "2.1.0"
)

//go:generate go run ../tools/template/main.go -name versionsTemplate versions.tf.template

func versions(dirname string, configurationAliases []ProviderAlias) error {
	pathname := filepath.Join(dirname, "versions.tf")
	b, err := fileutil.ReadFile(pathname)
	log.Print(err)

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
			ArchiveVersion, AWSVersion, ExternalVersion string
			ConfigurationAliases                        []ProviderAlias
			TerraformVersion                            string
		}{
			archiveVersion, awsVersion, externalVersion,
			configurationAliases,
			TerraformVersion,
		})
	}

	// Use crude but at least precise regular expressions to upgrade provider
	// versions as necessary without disturbing additional providers that
	// folks may add.

	b = regexp.MustCompile(
		`# managed by Substrate; do not edit by hand`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`# partially managed by Substrate; do not edit the archive, aws, or external providers by hand`,
		TerraformVersion,
	)))

	b = regexp.MustCompile(
		`source\s+=\s+"hashicorp/archive"
\s+version\s+=\s+">?(= )?\d+\.\d+\.\d+"`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`source = "hashicorp/archive"
      version = ">= %s"`,
		archiveVersion,
	)))

	b = regexp.MustCompile(
		`source\s+=\s+"hashicorp/aws"
\s+version\s+=\s+">?(= )?\d+\.\d+\.\d+"`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`source = "hashicorp/aws"
      version = ">= %s"`,
		awsVersion,
	)))
	// TODO also might need to handle configuration_aliases for completeness (but don't need to this month)

	b = regexp.MustCompile(
		`source\s+=\s+"hashicorp/external"
\s+version\s+=\s+">?(= )?\d+\.\d+\.\d+"`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`source = "hashicorp/external"
      version = ">= %s"`,
		externalVersion,
	)))

	b = regexp.MustCompile(
		`required_version\s+=\s+">?(= )?\d+\.\d+\.\d+"`,
	).ReplaceAllLiteral(b, []byte(fmt.Sprintf(
		`required_version = "= %s"`,
		TerraformVersion,
	)))

	return ioutil.WriteFile(pathname, b, 0666)
}
