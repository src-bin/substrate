package choices

import (
	"log"
	"os"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/ui"
)

const (
	CloudTrailRegionFilename = "substrate.CloudTrail-region"
	DefaultRegionFilename    = "substrate.default-region"
	PrefixFilename           = "substrate.prefix"
)

func DefaultRegion() string {

	// Migrate from the former filename if possible.
	if fileutil.Exists(CloudTrailRegionFilename) {
		if fileutil.Exists(DefaultRegionFilename) {
			ui.Printf(
				"could not migrate %s to %s because %s already exists",
				CloudTrailRegionFilename,
				DefaultRegionFilename,
				DefaultRegionFilename,
			)
		} else {
			if err := os.Rename(CloudTrailRegionFilename, DefaultRegionFilename); nil != err {
				log.Fatal(err)
			}
		}
	}

	region, err := ui.PromptFile(
		DefaultRegionFilename,
		"what region is your default for hosting e.g. the S3 buckets that stores your CloudTrail logs or Terraform state?",
	)
	if err != nil {
		log.Fatal(err)
	}
	if !regions.IsRegion(region) {
		log.Fatalf("%s is not an AWS region", region)
	}
	ui.Printf("using region %s", region)
	return region
}

func Prefix() string {
	prefix, err := ui.PromptFile(
		PrefixFilename,
		"what prefix do you want to use for global names like S3 buckets? (Substrate recommends your company name, all lower case)",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using prefix %s", prefix)
	return prefix
}
