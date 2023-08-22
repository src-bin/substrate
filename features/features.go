package features

import (
	"os"
	"strings"
)

func Enabled(feature string) bool {
	for _, f := range strings.Split(os.Getenv("SUBSTRATE_FEATURES"), ",") {
		if feature == f {
			return true
		}
	}
	return false
}
