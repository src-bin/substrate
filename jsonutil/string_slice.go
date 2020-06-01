package jsonutil

import (
	"encoding/json"
	"sort"
)

// StringSlice is a JSON-aware []string that can cope with AWS' bad habit of
// turning single-element JSON arrays of strings into strings.
type StringSlice []string

// Add adds the given string to p as if it were a set.  That is, s is added
// to p if and only if it isn't already there.  p will be sorted every time
// this method is called but it will usually be a no-op.
func (p *StringSlice) Add(s string) {
	ss := *p
	sort.Sort(ss)
	i := sort.SearchStrings(ss, s)
	if i == len(ss) || (ss)[i] != s { // s is not in ss
		ss_ := make(StringSlice, len(ss)+1)
		copy(ss_[:i], ss[:i])
		ss_[i] = s
		copy(ss_[i+1:], ss[i:])
		*p = ss_
	}
}

func (ss StringSlice) Len() int { return len(ss) }

func (ss StringSlice) Less(i, j int) bool { return ss[i] < ss[j] }

// MarshalJSON is unnecessary because it's always valid to marshal this type
// as a []string usually would be.
/*
func (ss StringSlice) MarshalJSON() ([]byte, error) {
}
*/

func (ss StringSlice) Swap(i, j int) {
	tmp := ss[i]
	ss[i] = ss[j]
	ss[j] = tmp
}

func (p *StringSlice) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*p = StringSlice{s}
		return nil
	}
	var ss []string
	if err := json.Unmarshal(b, &ss); err != nil {
		return err
	}
	*p = StringSlice(ss)
	return nil
}
