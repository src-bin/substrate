package cmdutil

import (
	"flag"
	"io/fs"

	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/ui"
)

func DomainFlag(help string) *string {
	return flag.String("domain", "", help)
}

func EnvironmentFlag(help string) *string {
	return flag.String("environment", "", help)
}

func QualityFlag(help string) *string {
	qualities, err := naming.Qualities()
	if err != nil && err == fs.ErrNotExist {
		ui.Fatal(err)
	}
	quality := ""
	if len(qualities) == 1 {
		quality = qualities[0]
	}
	return flag.String("quality", quality, help)
}
