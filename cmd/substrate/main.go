package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/src-bin/substrate/ui"
)

//go:generate go run ../../tools/dispatch-map/main.go -package main

// includeInDispatchMap, when declared as the only argument to a package-level
// function, signals the dispatch-map generator to include that function in the
// dispatch map.
type includeInDispatchMap struct{}

func main() {

	// If we were invoked directly, expect to find a subcommand in the first
	// position. Reconfigure the arguments to make it look like we were invoked
	// via a symbolic link so as to unify dispatch and simply argument parsing
	// after dispatch.
	executable, err := os.Executable()
	if err != nil {
		ui.Fatal(err)
	}
	if filepath.Base(os.Args[0]) == filepath.Base(executable) {
		if len(os.Args) < 2 {
			ui.Fatal("not enough arguments")
		}
		os.Args = append([]string{fmt.Sprintf("substrate-%s", os.Args[1])}, os.Args[2:]...)
	}

	// Dispatch to the function in this package named like the subcommand.
	f, ok := dispatchMap[strings.TrimPrefix(os.Args[0], "substrate-")]
	if !ok {
		ui.Fatalf("%s not found", os.Args[0]) // possibly confusing if invoked as a subcommand
	}
	f(includeInDispatchMap{})

}
