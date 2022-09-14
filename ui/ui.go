package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/src-bin/substrate/version"
	"golang.org/x/crypto/ssh/terminal"
)

func Confirm(args ...interface{}) (bool, error) {
	for {
		yesno, err := Prompt(args...)
		if err != nil {
			return false, err
		}
		if strings.ToLower(yesno) == "yes" {
			return true, nil
		}
		if strings.ToLower(yesno) == "no" {
			return false, nil
		}
		Print(`please respond "yes" or "no"`)
	}
}

func Confirmf(format string, args ...interface{}) (bool, error) {
	return Confirm(fmt.Sprintf(format, args...))
}

func Fatal(args ...interface{}) {
	args = dereference(args)
	op(opFatal, fmt.Sprint(withCaller(args...)...))
}

func Fatalf(format string, args ...interface{}) {
	args = dereference(args)
	op(opFatal, fmt.Sprintf(format, withCaller(args...)...))
}

func Print(args ...interface{}) {
	args = dereference(args)
	op(opPrint, fmt.Sprint(args...))
}

func PrintWithCaller(args ...interface{}) {
	Print(withCaller(args...)...)
}

func Printf(format string, args ...interface{}) {
	args = dereference(args)
	op(opPrint, fmt.Sprintf(format, args...))
}

func Prompt(args ...interface{}) (string, error) {
	op(opBlock, "")
	defer op(opUnblock, "")
	args = dereference(args)
	fmt.Fprint(os.Stderr, append(args, " ")...)
	if Interactivity() == NonInteractive {
		Fatal("(cannot accept input in non-interactive mode)")
	}
	s, err := stdin.ReadString('\n')
	if err != nil {
		return "", err
	}
	s = strings.Trim(s, "\r\n")
	if !terminal.IsTerminal(0) {
		fmt.Fprintf(os.Stderr, "%s (read from non-TTY)\n", s)
	}
	return s, nil
}

func Promptf(format string, args ...interface{}) (string, error) {
	return Prompt(fmt.Sprintf(format, args...))
}

func Spin(args ...interface{}) {
	args = dereference(args)
	op(opSpin, fmt.Sprint(args...))
}

func Spinf(format string, args ...interface{}) {
	args = dereference(args)
	op(opSpin, fmt.Sprintf(format, args...))
}

func Stop(args ...interface{}) {
	args = dereference(args)
	op(opStop, fmt.Sprint(args...))
}

func Stopf(format string, args ...interface{}) {
	args = dereference(args)
	op(opStop, fmt.Sprintf(format, args...))
}

func dereference(args []interface{}) []interface{} {
	returns := make([]interface{}, len(args))
	for i, arg := range args {
		if p, ok := arg.(*string); ok {
			if p != nil {
				returns[i] = *p
			} else {
				returns[i] = ""
			}
		} else {
			returns[i] = args[i]
		}
	}
	return returns
}

func shorten(pathname string) string {
	return filepath.Join(
		filepath.Base(filepath.Dir(pathname)),
		filepath.Base(pathname),
	)
}

// withCaller decorates log lines with caller information, though in a way that
// feels less to customers like they did something horrible. This is cribbed
// from the standard library's log.Logger.Output.
// <https://cs.opensource.google/go/go/+/refs/tags/go1.18.3:src/log/log.go;l=172>
func withCaller(args ...interface{}) []interface{} {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		fatal := fmt.Sprintf("%s:%d", shorten(file), line)
		_, file, line, ok = runtime.Caller(3)
		if ok {
			args = append(args, fmt.Sprintf(
				" (%s via %s:%d; Substrate version %s)",
				fatal,
				shorten(file),
				line,
				version.Version,
			))
		} else {
			args = append(args, fmt.Sprintf(
				" (%s; Substrate version %s)",
				fatal,
				version.Version,
			))
		}
	}
	return args
}
