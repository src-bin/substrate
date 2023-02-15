package cmdutil

import (
	"flag"
	"testing"
)

func TestStringSliceFlagVar(t *testing.T) {
	test := StringSliceFlag("test", "test")
	testVar := []string{}
	StringSliceFlagVar(testVar, "test-var", "test-var")
	flag.CommandLine.Parse([]string{
		"-test", "foo",
		"-test", "bar",
		"-test-var", "foo",
		"-test-var", "bar",
	})
	t.Log(test)
	t.Log(testVar)
	//if !slices.Equal(test.Slice(), testVar.Slice()) { // packages slices isn't in the standard library yet
	if test.Len() != len(testVar) {
		t.Fatalf("%v != %v", test.Slice(), testVar)
	}
	testSlice := test.Slice()
	for i := 0; i < len(testSlice); i++ {
		if testSlice[i] != testVar[i] {
			t.Fatalf("%v != %v", testSlice, testVar)
		}
	}
}
