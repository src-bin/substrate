package awsutil

import (
	"sort"
	"testing"
)

func TestBlacklistedRegions(t *testing.T) {
	if !sort.StringsAreSorted(BlacklistedRegions()) {
		t.Fatal("BlacklistedRegions() doesn't return a sorted []string")
	}
}
