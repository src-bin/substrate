package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/src-bin/substrate/ui"
)

//go:generate go run ../../tools/dispatch-map/main.go -package main

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

	// Dispatch to the package named like the subcommand with a Main function
	// or to the appropriate substrate-* program.
	f, ok := dispatchMap[strings.TrimPrefix(os.Args[0], "substrate-")]
	if !ok {
		if _, err := exec.LookPath(os.Args[0]); err != nil {
			ui.Fatalf("%s not found", os.Args[0])
		}
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1)
		}
	}
	f()

}
