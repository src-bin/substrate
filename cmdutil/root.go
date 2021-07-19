package cmdutil

import (
	"os"

	"github.com/src-bin/substrate/ui"
)

const substrateRoot = "SUBSTRATE_ROOT"

func Chdir() error {
	if dirname := os.Getenv(substrateRoot); dirname != "" {
		return os.Chdir(dirname)
	}
	return nil
}

func MustChdir() {
	if err := Chdir(); err != nil {
		ui.Fatal(err)
	}
}
