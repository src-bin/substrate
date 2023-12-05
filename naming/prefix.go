package naming

import (
	"os"
	"strings"

	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

const PrefixFilename = "substrate.prefix"

func Prefix() (prefix string) {

	// For Lambda, in the short-term, but perhaps eventually for other
	// clients that don't have the Substrate repo checked out or even a
	// future in which the Substrate repo becomes optional.
	if prefix = os.Getenv("SUBSTRATE_PREFIX"); prefix != "" {
		return
	}

	var err error
	for {
		prefix, err = ui.PromptFile(
			PrefixFilename,
			"what prefix do you want to use for global names like S3 buckets? (Substrate recommends your company name, all lower case)",
		)
		ui.Must(err)
		if prefix == "trial" {
			ui.Printf("%q is a reserved prefix; please something else", prefix)
			ui.Must(fileutil.Remove(PrefixFilename))
		} else {
			break
		}
	}

	if !printedPrefix {
		ui.Printf("using prefix %s", prefix)
		printedPrefix = true
	}
	return
}

func PrefixNoninteractive() (string, error) {
	pathname, err := fileutil.PathnameInParents(PrefixFilename)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(pathname)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(b), "\r\n"), nil
}

var printedPrefix bool
