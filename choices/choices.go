package choices

import (
	"log"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

const (
	CloudTrailRegionFilename      = "substrate.CloudTrail-region"
	DefaultRegionFilename         = "substrate.default-region"
	IntranetDNSDomainNameFilename = "substrate.intranet-dns-domain-name"
	PrefixFilename                = "substrate.prefix"
)

var printedDefaultRegion, printedPrefix bool

func DefaultRegion() string {
	region, err := ui.PromptFile(
		DefaultRegionFilename,
		"what region is your default for hosting shared resources e.g. the S3 bucket that stores your CloudTrail logs?",
	)
	if err != nil {
		log.Fatal(err)
	}
	if !regions.IsRegion(region) {
		log.Fatalf("%s is not an AWS region", region)
	}
	if !printedDefaultRegion {
		ui.Printf("using region %s as your default", region)
		printedDefaultRegion = true
	}
	return region
}

func DefaultRegionNoninteractive() string {
	b, err := fileutil.ReadFile(DefaultRegionFilename)
	if err != nil {
		ui.Fatal(err)
	}
	return fileutil.Tidy(b)
}

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
