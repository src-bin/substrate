package version

import (
	"flag"
	"fmt"
	"os"
)

func Flag() {
	if !flag.Parsed() {
		panic("version.Flag must be called after flag.Parse")
	}
	if *versionFlag {
		Print()
		os.Exit(0)
	}
}

func Print() {
	fmt.Fprintf(os.Stderr, "Substrate version %s\n", Version) // ui.Printf would be a dependency cycle
}

var Version = "1970.01" // replaced at build time with current computed version; see Makefile

var versionFlag = flag.Bool("version", false, "print Substrate version information and exit")
