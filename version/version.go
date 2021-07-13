package version

import (
	"flag"

	"github.com/src-bin/substrate/ui"
)

func Flag() {
	if !flag.Parsed() {
		panic("version.Flag must be called after flag.Parse")
	}
	if *versionFlag {
		ui.Fatalf("Substrate version %s", Version)
	}
}

var Version = "1970.01" // replaced at build time with current computed version; see Makefile

var versionFlag = flag.Bool("version", false, "print Substrate version information and exit")
