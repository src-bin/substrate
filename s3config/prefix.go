package s3config

import (
	"log"

	"github.com/src-bin/substrate/ui"
)

const PrefixFilename = "substrate.prefix"

func Prefix() string {
	prefix, err := ui.PromptFile(
		PrefixFilename,
		"what prefix do you want to use for global names like S3 buckets?",
	)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf("using prefix %s", prefix)
	return prefix
}
