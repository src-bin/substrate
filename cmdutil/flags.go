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

type StringSliceFlag []string

func StringSlice(name, usage string) *StringSliceFlag {
	ss := &StringSliceFlag{}
	flag.Var(ss, name, usage)
	return ss
}

func (ssf *StringSliceFlag) Set(s string) error {
	*ssf = append(*ssf, s)
	return nil
}

func (ssf *StringSliceFlag) Slice() []string {
	if ssf == nil || *ssf == nil {
		return []string{}
	}
	return append([]string{}, *ssf...)
}

func (ssf *StringSliceFlag) String() string {
	return strings.Join(*ssf, ", ")
}
