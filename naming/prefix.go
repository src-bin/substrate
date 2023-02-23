package naming

import "github.com/src-bin/substrate/ui"

const PrefixFilename = "substrate.prefix"

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

var printedPrefix bool
