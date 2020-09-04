package fileutil

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
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
	cmd.Stderr = os.Stderr
	//log.Printf("%+v", cmd)
	return cmd.Run()
}

func Exists(pathname string) bool {
	_, err := os.Stat(pathname)
	return err == nil
}

func FromLines(ss []string) []byte {
	return []byte(strings.Join(ss, "\n") + "\n")
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

// Tidy removes newline-like characters from either end of a []byte and
// returns the middle as a string.
func Tidy(b []byte) string {
	return strings.Trim(strings.Replace(string(b), "\r", "\n", -1), "\n")
}

func ToLines(b []byte) []string {
	return strings.Split(Tidy(b), "\n")
}
