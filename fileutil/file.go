package fileutil

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
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
	err := cmd.Run()
	if _, ok := err.(*exec.ExitError); ok {
		err = nil // because NeoVim often misbehaves and yet is very popular
	}
	return err
}

func Exists(pathname string) bool {
	_, err := os.Stat(pathname)
	return err == nil
}

func FromLines(ss []string) []byte {
	return []byte(strings.Join(ss, "\n") + "\n")
}

func IsDir(pathname string) bool {
	fi, err := os.Stat(pathname)
	return err == nil && fi.IsDir()
}

// NotEmpty returns true if the file at pathname exists and has at least one
// byte in it. This is written in the negative because, if the function were
// simply Empty(pathname), it would almost always need to be combined with
// Exists(pathname) and that would mean a superfluous second os.Stat(pathname).
func NotEmpty(pathname string) bool {
	fi, err := os.Stat(pathname)
	return err == nil && fi.Size() > 0
}

// PathnameInParents searches the current working directory and each of its
// parents for pathname. It returns the closest relative pathname in which the
// given filename exists or an error if filename doesn't exist in any parent.
func PathnameInParents(filename string) (string, error) {
	pathname := filename
	for {
		if Exists(pathname) {
			return pathname, nil
		}
		pathname = filepath.Join("..", pathname)
		if dirname, err := filepath.Abs(filepath.Dir(pathname)); err != nil {
			return "", err
		} else if dirname == "/" {
			break
		}
	}
	return "", fs.ErrNotExist
}

// Remove removes a file via os.Remove and returns every error it can return
// except fs.ErrNotExist, which it silences.
func Remove(pathname string) error {
	err := os.Remove(pathname)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// Tidy removes newline-like characters from either end of a []byte and
// returns the middle as a string.
func Tidy(b []byte) string {
	return strings.Trim(strings.Replace(string(b), "\r", "\n", -1), "\n")
}

func ToLines(b []byte) []string {
	return strings.Split(Tidy(b), "\n")
}

func WriteFileIfNotExists(pathname string, b []byte) error {
	f, err := os.OpenFile(pathname, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}
