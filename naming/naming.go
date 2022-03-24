package naming

import (
	"log"

	"github.com/src-bin/substrate/ui"
)

const (
	IntranetDNSDomainNameFilename = "substrate.intranet-dns-domain-name"
	PrefixFilename                = "substrate.prefix"
)

var printedPrefix bool

func Prefix() string {
	prefix, err := ui.PromptFile(
		PrefixFilename,
		"what prefix do you want to use for global names like S3 buckets? (Substrate recommends your company name, all lower case)",
	)
	if err != nil {
		log.Fatal(err)
	}
	if !printedPrefix {
		ui.Printf("using prefix %s", prefix)
		printedPrefix = true
	}
	return prefix
}
