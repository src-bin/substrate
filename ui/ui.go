package ui

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/src-bin/substrate/fileutil"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	opPrint = iota
	opSpin
	opStop
)

const hz = 10 // spins per second

type instruction struct {
	ch     chan<- struct{}
	opcode int
	s      string
}

var (
	chInst chan<- instruction
	stdin  *bufio.Reader
)

func Confirm(args ...interface{}) (string, error) {
	for {
		yesno, err := Prompt(args...)
		if err != nil {
			return "", err
		}
		if strings.ToLower(yesno) == "yes" {
			return "yes", nil
		}
		if strings.ToLower(yesno) == "no" {
			return "no", nil
		}
		Print(`please respond "yes" or "no"`)
	}
}

func Confirmf(format string, args ...interface{}) (string, error) {
	return Confirm(fmt.Sprintf(format, args...))
}

func Print(args ...interface{}) {
	dereference(args)
	op(opPrint, fmt.Sprint(args...))
}

func Printf(format string, args ...interface{}) {
	dereference(args)
	op(opPrint, fmt.Sprintf(format, args...))
}

func Prompt(args ...interface{}) (string, error) {
	dereference(args)
	fmt.Print(append(args, " ")...)
	s, err := stdin.ReadString('\n')
	if err != nil {
		return "", err
	}
	if !terminal.IsTerminal(0) {
		fmt.Println("(read from non-TTY)")
	}
	return strings.TrimSuffix(s, "\n"), nil
}

// PromptFile wraps Prompt in ReadFile and ioutil.WriteFile to avoid prompting
// at all on subsequent invocations.  If pathname exists, its contents
// (chomped) will be taken as the response to the prompt.  If not, the response
// to the prompt is written to pathname with a trailing newline and a notice is
// printed instructing the user to commit that file to version control.
func PromptFile(pathname string, args ...interface{}) (string, error) {
	buf, err := fileutil.ReadFile(pathname)
	s := strings.TrimSuffix(string(buf), "\n")
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

func Promptf(format string, args ...interface{}) (string, error) {
	return Prompt(fmt.Sprintf(format, args...))
}

// PromptfFile is like PromptFile but allows formatting of the prompt.
func PromptfFile(pathname, format string, args ...interface{}) (string, error) {
	return PromptFile(pathname, fmt.Sprintf(format, args...))
}

func Spin(args ...interface{}) {
	dereference(args)
	op(opSpin, fmt.Sprint(args...))
}

func Spinf(format string, args ...interface{}) {
	dereference(args)
	op(opSpin, fmt.Sprintf(format, args...))
}

func Stop(args ...interface{}) {
	dereference(args)
	op(opStop, fmt.Sprint(args...))
}

func Stopf(format string, args ...interface{}) {
	dereference(args)
	op(opStop, fmt.Sprintf(format, args...))
}

// init starts a goroutine that serializes all ui elements so that the
// terminal's output makes sense.
//
// Yes, I know init functions are unsavory and, yes, I know that the package
// variables they imply are probably worse.  This is useful, though.
func init() {
	ch := make(chan instruction)
	chInst = ch
	stdin = bufio.NewReader(os.Stdin)
	go func(ch <-chan instruction) {
		dots, s, spinner := "", "", ""
		tick := time.Tick(time.Second / hz)
		ticks := 1
		for {
			isTerminal := terminal.IsTerminal(1)
			width, _, err := terminal.GetSize(1)
			if err != nil {
				width = 80
			}

			select {

			case inst := <-ch:
				switch inst.opcode {
				case opPrint:

					// Print called between Spin and Stop
					// demands special consideration.
					if spinner != "" {
						if isTerminal {
							fmt.Print("\r", s, " ", dots, ". (to be continued)\n")
						} else {
							fmt.Println(" (to be continued)")
						}
						dots, s = "", "(continuing)"
					}

					fmt.Println(inst.s)

					// Per above, indicate that the spinning is resuming.
					if spinner != "" {
						fmt.Print("(continuing)")
					}

				case opSpin:

					// The last line of output on the terminal can't wrap or
					// carriage returns will make a mess of things.
					var i int
					if isTerminal {
						i = len(inst.s) - len(inst.s)%width
						if i > 0 {
							fmt.Println(inst.s[:i])
						}
					}
					s, spinner = inst.s[i:], "-"
					fmt.Print(s, " ", dots, spinner)

				case opStop:

					// No carriage returns if standard output is not a terminal.
					if !isTerminal {
						fmt.Print(" ", strings.TrimSuffix(inst.s, "\n"), "\n")
						break
					}

					fmt.Print("\r", s, " ", dots, ". ", strings.TrimSuffix(inst.s, "\n"), "\n")
					dots, s, spinner = "", "", ""
				}
				inst.ch <- struct{}{}

			case <-tick:

				// No carriage returns if standard output is not a terminal.
				if !isTerminal {
					continue
				}

				if ticks%(2*hz) == 0 {
					dots = dots + "."
				}
				if spinner != "" {

					// If the spinner is about to wrap, output a newline and
					// align it to continue below.
					if len(fmt.Sprint("\r", s, " ", dots)) > width {
						fmt.Print("\r", s, " ", dots, "\n")
						dots, s = "", strings.Repeat(" ", len(s))
					}

					fmt.Print("\r", s, " ", dots, spinner)
				}
				switch spinner {
				case "-":
					spinner = "\\"
				case "\\":
					spinner = "|"
				case "|":
					spinner = "/"
				case "/":
					spinner = "-"
				}

				ticks = (ticks + 1) % (2 * hz)

			}
		}
	}(ch)
}

func dereference(args []interface{}) {
	for i, arg := range args {
		if p, ok := arg.(*string); ok {
			args[i] = *p
		}
	}
}

func op(opcode int, s string) {
	ch := make(chan struct{})
	chInst <- instruction{
		ch:     ch,
		opcode: opcode,
		s:      s,
	}
	<-ch
}
