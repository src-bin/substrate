package regions

import (
	"sort"
	"testing"
)

func TestRegionsBeingAvoided(t *testing.T) {
	if !sort.StringsAreSorted(Avoiding()) {
		t.Fatal("Avoiding() doesn't return a sorted []string")
	}
}
