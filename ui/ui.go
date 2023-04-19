package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/src-bin/substrate/awsutil"
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
	for i, arg := range args {
		if err, ok := arg.(error); ok {
			args[i] = helpful(err)
		}
	}
	op(opFatal, fmt.Sprint(withCaller(args...)...))
}

func Fatalf(format string, args ...interface{}) {
	args = dereference(args)
	for i, arg := range args {
		if err, ok := arg.(error); ok {
			args[i] = helpful(err)
		}
	}
	op(opFatal, fmt.Sprint(withCaller(fmt.Sprintf(format, args...))...))
}

func Must(err error) {
	if err != nil {
		op(opFatal, fmt.Sprint(withCaller(helpful(err))...))
	}
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

// StopErr calls Stop with either the error code from the given non-nil error as an argument or
// with the string "ok" otherwise.
func StopErr(err error) error {
	s := "ok"
	if err != nil {
		s = awsutil.ErrorCode(err)
		if s == "" {
			s = err.Error()
		}
	}
	Stop(s)
	return err
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

// helpful might swap an obtuse error for one that's more helpful so that the
// fatal error that's about to terminate the program can be...helpful.
func helpful(err error) error {

	// If the AWS SDK thinks it's missing a region it is almost certainly the
	// case that the program's been invoked outside the Substrate repository
	// and without SUBSTRATE_ROOT set.
	var mrErr *aws.MissingRegionError
	if errors.As(err, &mrErr) {
		return errors.New("couldn't find your default AWS region which most likely means this program's been invoked from outside your Substrate repository without SUBSTRATE_ROOT in the environment; change your working directory to your Substrate repository or set SUBSTRATE_ROOT in your environment to the absolute path to your Substrate repository")
	}

	// If the AWS SDK reports a signing error the most likely explanation is
	// that there aren't any AWS credentials in the environment.
	var sErr *v4.SigningError
	if errors.As(err, &sErr) {
		return fmt.Errorf("%w\ncouldn't find AWS credentials in the environment; you can most likely fix this by running `eval $(substrate credentials)`", err)
	}

	return err
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
