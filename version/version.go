package version

import "fmt"

func Print() {
	fmt.Printf( // ui.Printf would be a dependency cycle
		"Substrate version %s\n",
		Version,
	)
}

var Version = "1970.01" // replaced at build time with current computed version; see Makefile
