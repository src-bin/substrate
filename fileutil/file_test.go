package fileutil

import (
	"bytes"
	"os"
	"testing"
)

func TestPathnameInParents(t *testing.T) {
	if pathname, err := PathnameInParents("file_test.go"); err != nil || pathname != "file_test.go" {
		t.Error(pathname, err)
	}
	if pathname, err := PathnameInParents("Makefile"); err != nil || pathname != "../Makefile" {
		t.Error(pathname, err)
	}
}

func TestEdit(t *testing.T) {
	const filename = "TestEdit"
	if err := Remove(filename); err != nil {
		t.Fatal(err)
	}
	defer Remove(filename)

	os.Setenv("EDITOR", "./editor.sh")
	os.Setenv("EDITOR_TEXT", "foo")
	os.Unsetenv("EDITOR_EXIT_STATUS")
	if err := Edit(filename); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte("foo\n")) {
		t.Fatalf("%#v", string(b))
	}
}

func TestEditMisbehavedEditor(t *testing.T) {
	const filename = "TestEditMisbehavedEditor"
	if err := Remove(filename); err != nil {
		t.Fatal(err)
	}
	defer Remove(filename)

	os.Setenv("EDITOR", "./editor.sh")
	os.Setenv("EDITOR_TEXT", "foo")
	os.Setenv("EDITOR_EXIT_STATUS", "1")
	if err := Edit(filename); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, []byte("foo\n")) {
		t.Fatalf("%#v", string(b))
	}
}

func TestWriteFileIfNotExists(t *testing.T) {
	const filename = "TestWriteFileIfNotExists"
	if err := Remove(filename); err != nil {
		t.Fatal(err)
	}
	defer Remove(filename)

	for _, write := range [][]byte{
		[]byte("foo\n"), // we'll create the file and write this
		[]byte("bar\n"), // it will already exist so we won't write this
	} {
		if err := WriteFileIfNotExists(filename, write); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, []byte("foo\n")) {
			t.Fatalf("%#v", string(b))
		}
	}
}
