package fileutil

import (
	"testing"
)

func TestPathnameInParents(t *testing.T) {
	if pathname, err := PathnameInParents("file_test.go"); err != nil || pathname != "file_test.go" {
		t.Error(pathname, err)
	}
	if pathname, err := PathnameInParents("Makefile"); err != nil || pathname != "../Makefile" {
		t.Error(pathname, err)
	}
	if pathname, err := PathnameInParents("foo/bar/Makefile"); err != nil || pathname != "../../../foo/bar/Makefile" {
		t.Error(pathname, err)
	}
}
