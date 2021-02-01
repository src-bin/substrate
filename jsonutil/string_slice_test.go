package jsonutil

import "testing"

func TestStringSliceAdd(t *testing.T) {
	testStringSliceAdd(t, StringSlice{"3", "2", "1"}, "0")
	testStringSliceAdd(t, StringSlice{"3", "2", "0"}, "1")
	testStringSliceAdd(t, StringSlice{"3", "1", "0"}, "2")
	testStringSliceAdd(t, StringSlice{"2", "1", "0"}, "3")
}

func testStringSliceAdd(t *testing.T, ss StringSlice, s string) {
	ss.Add(s)
	ss.Add(s)
	if len(ss) != 4 || ss[0] != "0" || ss[1] != "1" || ss[2] != "2" || ss[3] != "3" {
		t.Error(ss, s)
	}
}
