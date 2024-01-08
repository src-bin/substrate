package version

import "fmt"

func Print() {
	var s string
	if Commit == Version {
		s = fmt.Sprintf("Substrate version %s\n", Version)
	} else {
		s = fmt.Sprintf("Substrate version %s (%s)\n", Version, Commit)
	}
	fmt.Print(s) // ui.Print would be a dependency cycle
}

var (
	Commit  = "0000000" // replaced at build time with the commit SHA that was built; see Makefile
	Version = "1970.01" // replaced at build time with tagged version or commit SHA; see Makefile
)
