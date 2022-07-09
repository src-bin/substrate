package regions

import (
	"sort"
	"testing"
)

func TestAllRegions(t *testing.T) {
	if len(allRegions) == 0 {
		t.Fatal("allRegions is empty")
	}
	if !sort.StringsAreSorted(allRegions) {
		t.Fatal("allRegions isn't sorted")
	}
}

func TestRegionsBeingAvoided(t *testing.T) {
	if !sort.StringsAreSorted(Avoiding()) {
		t.Fatal("Avoiding() doesn't return a sorted []string")
	}
}
