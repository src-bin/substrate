package versionutil

import "testing"

func TestVersionComparison(t *testing.T) {
	dirtyVersion := "2022.07.13.22.14.47" // for example
	newerVersion := "2022.07"
	newererVersion := "2022.08"
	olderVersion := "2022.06"
	if cmp := Compare(olderVersion, dirtyVersion); cmp != Less {
		t.Errorf("Compare(%q, %q): %v", olderVersion, dirtyVersion, cmp)
	}
	if cmp := Compare(dirtyVersion, newerVersion); cmp != Equal {
		t.Errorf("Compare(%q, %q): %v", dirtyVersion, newerVersion, cmp)
	}
	if cmp := Compare(dirtyVersion, newererVersion); cmp != Less {
		t.Errorf("Compare(%q, %q): %v", dirtyVersion, newererVersion, cmp)
	}
}
