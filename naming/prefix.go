package naming

import (
	"os"

	"github.com/src-bin/substrate/ui"
)

const PrefixFilename = "substrate.prefix"

func Prefix() string {

	// For Lambda, in the short-term, but perhaps eventually for other
	// clients that don't have the Substrate repo checked out or even a
	// future in which the Substrate repo becomes optional.
	if prefix := os.Getenv("SUBSTRATE_PREFIX"); prefix != "" {
		return prefix
	}

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
