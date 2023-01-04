package versionutil

import "testing"

func TestVersionComparison(t *testing.T) {
	dirtyVersion := "2022.07.13.22.14.47" // for example
	olderVersion := "2022.06"
	currentVersion := "2022.07"
	newerVersion := "2022.08"
	if cmp := Compare(olderVersion, dirtyVersion); cmp != Less {
		t.Errorf("Compare(%q, %q): %v", olderVersion, dirtyVersion, cmp)
	}
	if cmp := Compare(dirtyVersion, currentVersion); cmp != Equal {
		t.Errorf("Compare(%q, %q): %v", dirtyVersion, currentVersion, cmp)
	}
	if cmp := Compare(dirtyVersion, newerVersion); cmp != Less {
		t.Errorf("Compare(%q, %q): %v", dirtyVersion, newerVersion, cmp)
	}
}
