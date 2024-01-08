package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/src-bin/substrate/fileutil"
)

// ConfirmFile wraps Confirm in ReadFile and os.WriteFile to avoid
// interactivity on subsequent invocations. If pathname exists, its contents
// (chomped) will be taken as the confirmation.  If not, the confirmation given
// is written to pathname with a trailing newline and a notice is printed
// instructing the user to commit that file to version control.
func ConfirmFile(pathname string, args ...interface{}) (bool, error) {
	b, err := os.ReadFile(pathname)
	yesno := strings.ToLower(strings.Trim(string(b), "\r\n"))
	if err == nil {
		if yesno == "yes" {
			return true, nil
		} else if yesno == "no" {
			return false, nil
		}
		Printf("invalid yes/no %q read from %s", yesno, pathname)
	}
	if len(args) == 0 {
		return false, nil
	}
	ok, err := Confirm(args...)
	if err != nil {
		return false, err
	}
	if ok {
		yesno = "yes"
	} else {
		yesno = "no"
	}
	if err := os.WriteFile(pathname, []byte(yesno+"\n"), 0666); err != nil {
		return false, err
	}
	Printf("%q written to %s, which you should commit to version control", yesno, pathname)
	return ok, nil
}

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
		b, err := os.ReadFile(pathname)
		if errors.Is(err, fs.ErrNotExist) {
			b = []byte("")
			err = nil
		}
		if err != nil {
			return nil, err
		}
		if len(b) != 0 {
			Print(notice)
			lines := fileutil.ToLines(b)
			for _, s := range lines {
				Printf("\t%s", s)
			}
			if Interactivity() < FullyInteractive {
				Print("if this is not correct, press ^C and re-run this command with --fully-interactive")
				time.Sleep(5e9) // give them a chance to ^C
				return lines, nil
			}
			ok, err := Confirm("is this correct? (yes/no)")
			if err != nil {
				return nil, err
			}
			if ok {
				return lines, nil
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
		if !fileutil.NotEmpty(pathname) {
			return nil, fmt.Errorf("nothing written to %s", pathname)
		}
		Printf("wrote %s, which you should commit to version control", pathname)
	}
}

// PromptFile wraps Prompt in ReadFile and os.WriteFile to avoid prompting
// at all on subsequent invocations.  If pathname exists, its contents
// (chomped) will be taken as the response to the prompt.  If not, the response
// to the prompt is written to pathname with a trailing newline and a notice is
// printed instructing the user to commit that file to version control.
func PromptFile(pathname string, args ...interface{}) (string, error) {
	b, err := os.ReadFile(pathname)
	s := strings.Trim(string(b), "\r\n")
	if err != nil {
		if len(args) == 0 {
			return s, nil
		}
		s, err = Prompt(args...)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(pathname, []byte(s+"\n"), 0666); err != nil {
			return "", err
		}
		Printf("%q written to %s, which you should commit to version control", s, pathname)
	}
	return s, nil
}

// PromptfFile is like PromptFile but allows formatting of the prompt.
func PromptfFile(pathname, format string, args ...interface{}) (string, error) {
	return PromptFile(pathname, fmt.Sprintf(format, args...))
}
