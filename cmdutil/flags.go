package cmdutil

import (
	"flag"
	"strings"

	"github.com/src-bin/substrate/naming"
)

func DomainFlag(help string) *string {
	return flag.String("domain", "", help)
}

func EnvironmentFlag(help string) *string {
	return flag.String("environment", "", help)
}

func QualityFlag(help string) *string {
	quality := ""
	if qualities, _ := naming.Qualities(); len(qualities) == 1 {
		quality = qualities[0]
	}
	return flag.String("quality", quality, usage)
}

type StringSlice []string

func StringSliceFlag(name, usage string) *StringSlice {
	ss := &StringSlice{}
	flag.Var(ss, name, usage)
	return ss
}

func (ss *StringSlice) String() string {
	return strings.Join(*ss, ", ")
}

func (ss *StringSlice) Set(s string) error {
	*ss = append(*ss, s)
	return nil
}
