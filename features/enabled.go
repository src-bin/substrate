package features

import (
	"os"
	"strings"
)

const EnvironmentVariable = "SUBSTRATE_FEATURES"

// feature attaches the Enabled method to strings so that feature flags may
// be checked in code with features.FlagName.Enabled(). This type is not
// exported to force feature flags to be defined in this package. They're in
// the features.go file.
type feature string

// Enabled returns true if the string value of the receiver appears in the
// comma-delimited SUBSTRATE_FEATURES environment variable and false otherwise.
func (f feature) Enabled() bool {
	featureName := string(f)
	for _, envName := range strings.Split(os.Getenv(EnvironmentVariable), ",") {
		if envName == featureName {
			return true
		}
	}
	return false
}
