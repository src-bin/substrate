package cmdutil

import (
	"os"
)

const substrateRoot = "SUBSTRATE_ROOT"

var oldpwd string // breadcrumb for UndoChdir

// Chdir changes the working directory to the value of the SUBSTRATE_ROOT
// environment variable, if set and non-empty. It returns the previous working
// directory and the error returned by os.Chdir.
func Chdir() (err error) {
	oldpwd, err = os.Getwd()
	if err != nil {
		return
	}
	if dirname := os.Getenv(substrateRoot); dirname != "" {
		err = os.Chdir(dirname)
	}
	return
}

// UndoChdir changes the working directory to whatever the working directory
// was before a prior call to Chdir. It panics if Chdir hasn't been called.
func UndoChdir() error {
	if oldpwd == "" {
		panic("UndoChdir called before Chdir or MustChdir")
	}
	err := os.Chdir(oldpwd)
	oldpwd = ""
	return err
}
