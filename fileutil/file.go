package fileutil

import (
	"io/ioutil"
	"os"
	"os/exec"
)

const DefaultEditor = "vim"

func Edit(pathname string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = DefaultEditor
	}
	cmd := exec.Command(editor, pathname)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	//log.Printf("%+v", cmd)
	return cmd.Run()
}

// ReadFile is ioutil.WriteFile's brother from another mother.
func ReadFile(pathname string) ([]byte, error) {
	f, err := os.Open(pathname)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(f)
	if err := f.Close(); err != nil {
		return nil, err
	}
	return b, err
}
