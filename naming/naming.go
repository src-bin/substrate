package naming

import (
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

const (
	EnvironmentsFilename = "substrate.environments"
	QualitiesFilename    = "substrate.qualities"

	IntranetDNSDomainNameFilename = "substrate.intranet-dns-domain-name"

	PrefixFilename = "substrate.prefix"
)

func Environments() ([]string, error) {
	b, err := fileutil.ReadFile(EnvironmentsFilename)
	if err != nil {
		return nil, err
	}
	return fileutil.ToLines(b), nil
}

func Prefix() string {
	prefix, err := ui.PromptFile(
		PrefixFilename,
		"what prefix do you want to use for global names like S3 buckets? (Substrate recommends your company name, all lower case)",
	)
	if err != nil {
		ui.Fatal(err)
	}
	if !printedPrefix {
		ui.Printf("using prefix %s", prefix)
		printedPrefix = true
	}
	return prefix
}

func Qualities() ([]string, error) {
	b, err := fileutil.ReadFile(QualitiesFilename)
	if err != nil {
		return nil, err
	}
	return fileutil.ToLines(b), nil
}

var printedPrefix bool
