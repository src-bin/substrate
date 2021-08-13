package cmdutil

import (
	"os"

	"github.com/src-bin/substrate/ui"
)

const substrateRoot = "SUBSTRATE_ROOT"

// Chdir changes the working directory to the value of the SUBSTRATE_ROOT
// environment variable, if set and non-empty. It returns the previous working
// directory and the error returned by os.Chdir.
func Chdir() (oldpwd string, err error) {
	oldpwd, err = os.Getwd()
	if err != nil {
		return
	}
	if dirname := os.Getenv(substrateRoot); dirname != "" {
		err = os.Chdir(dirname)
	}
	return
}

// MustChdir calls Chdir and terminates the process if it returns an error.
func MustChdir() string {
	oldpwd, err := Chdir()
	if err != nil {
		ui.Fatal(err)
	}
	return oldpwd
}
