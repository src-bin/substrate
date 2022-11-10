package cmdutil

import (
	"flag"
	"io/fs"
	"log"

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
	log.Print(qualities, err)
	if err != nil && err == fs.ErrNotExist {
		ui.Fatal(err)
	}
	quality := ""
	if len(qualities) == 1 {
		quality = qualities[0]
		ui.Print("you only have one quality, so we're defaulting to it - ", quality)
	}
	return flag.String("quality", quality, help)
}
