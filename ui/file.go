package ui

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/src-bin/substrate/fileutil"
)

// EditFile guides a user to edit a plaintext file to provide input.  If the
// file exists, it is first read and the user is offered the opportunity to
// accept it without modification via the notice argument to this function.  If
// the file doesn't exist, or the user asks to edit its contents, their EDITOR
// (or vim, as a default) is opened and they're given instructions via the
// instructions argument to this function.  After the file's written and the
// editor exits, the file is read once more and its final contents is returned
// as a slice of strings, each representing a line in the file (without the
// trailing newline).
func EditFile(pathname, notice, instructions string) ([]string, error) {
	for {
		b, err := fileutil.ReadFile(pathname)
		if errors.Is(err, os.ErrNotExist) {
			b = []byte("")
			err = nil
		}
		if err != nil {
			return nil, err
		}
		if len(b) != 0 {
			Print(notice)
			for _, s := range fileutil.ToLines(b) {
				Printf("\t%s", s)
			}
			ok, err := Confirm("is this correct? (yes/no)")
			if err != nil {
				return nil, err
			}
			if ok {
				return fileutil.ToLines(b), nil
			}
		}
		if _, err := Promptf(
			"press <enter> to open your EDITOR; %s; save and exit when you're finished",
			instructions,
		); err != nil {
			return nil, err
		}
		if err := fileutil.Edit(pathname); err != nil {
			return nil, err
		}
		Printf("wrote %s, which you should commit to version control", pathname)
	}
}

// PromptFile wraps Prompt in ReadFile and ioutil.WriteFile to avoid prompting
// at all on subsequent invocations.  If pathname exists, its contents
// (chomped) will be taken as the response to the prompt.  If not, the response
// to the prompt is written to pathname with a trailing newline and a notice is
// printed instructing the user to commit that file to version control.
func PromptFile(pathname string, args ...interface{}) (string, error) {
	b, err := fileutil.ReadFile(pathname)
	s := strings.Trim(string(b), "\r\n")
	if err != nil {
		s, err = Prompt(args...)
		if err != nil {
			return "", err
		}
		if err := ioutil.WriteFile(pathname, []byte(s+"\n"), 0666); err != nil {
			return "", err
		}
		Printf("\"%s\" written to %s, which you should commit to version control", s, pathname)
	}
	return s, nil
}

// PromptfFile is like PromptFile but allows formatting of the prompt.
func PromptfFile(pathname, format string, args ...interface{}) (string, error) {
	return PromptFile(pathname, fmt.Sprintf(format, args...))
}
