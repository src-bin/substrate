package cmdutil

import (
	"flag"
	"strings"

	"github.com/src-bin/substrate/naming"
)

func DomainFlag(usage string) *string {
	return flag.String("domain", "", usage)
}

func EnvironmentFlag(usage string) *string {
	return flag.String("environment", "", usage)
}

func QualityFlag(usage string) *string {
	quality := ""
	if qualities, _ := naming.Qualities(); len(qualities) == 1 {
		quality = qualities[0]
	}
	return flag.String("quality", quality, usage)
}

type StringSlice struct {
	slice []string
}

func StringSliceFlag(name, usage string) *StringSlice {
	return StringSliceFlagVar([]string{}, name, usage)
}

func StringSliceFlagVar(slice []string, name, usage string) *StringSlice {
	ss := &StringSlice{slice}
	flag.Var(ss, name, usage)
	return ss
}

func (ss *StringSlice) Len() int { return len(ss.slice) }

func (ss *StringSlice) Slice() []string { return ss.slice }

func (ss *StringSlice) String() string { return strings.Join(ss.slice, ", ") }

func (ss *StringSlice) Set(s string) error {
	ss.slice = append(ss.slice, s)
	return nil
}
