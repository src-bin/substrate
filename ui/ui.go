package ui

import (
	"fmt"
	"os"
	"strings"

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
	Print(args...)
	os.Exit(1)
}

func Fatalf(format string, args ...interface{}) {
	Printf(format, args...)
	os.Exit(1)
}

func Print(args ...interface{}) {
	args = dereference(args)
	op(opPrint, fmt.Sprint(args...))
}

func Printf(format string, args ...interface{}) {
	args = dereference(args)
	op(opPrint, fmt.Sprintf(format, args...))
}

func Prompt(args ...interface{}) (string, error) {
	op(opBlock, "")
	defer op(opUnblock, "")
	args = dereference(args)
	fmt.Print(append(args, " ")...)
	if Interactivity() == NonInteractive {
		Fatal("(cannot accept input in non-interactive mode)")
	}
	s, err := stdin.ReadString('\n')
	if err != nil {
		return "", err
	}
	s = strings.Trim(s, "\r\n")
	if !terminal.IsTerminal(0) {
		fmt.Printf("%s (read from non-TTY)\n", s)
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
