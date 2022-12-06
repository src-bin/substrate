package cmdutil

import (
	"flag"

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
	return flag.String("quality", quality, help)
}
