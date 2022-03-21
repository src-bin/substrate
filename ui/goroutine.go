package ui

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	opPrint = iota
	opQuiet
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

// init starts a goroutine that serializes all ui elements so that the
// terminal's output makes sense.
//
// Yes, I know init functions are unsavory and, yes, I know that the package
// variables they imply are probably worse.  This is useful, though.
func init() {
	log.SetFlags(log.Lshortfile)
	ch := make(chan instruction)
	chInst = ch
	stdin = bufio.NewReader(os.Stdin)
	stderr := os.Stderr
	go func(ch <-chan instruction) {
		dots, s, spinner := "", "", ""
		tick := time.Tick(time.Second / hz)
		ticks := 1
		for {
			isTerminal := terminal.IsTerminal(2)
			width, _, err := terminal.GetSize(2)
			if err != nil {
				width = 80
			}
			if width == 0 {
				isTerminal = false
			}

			select {

			case inst := <-ch:
				switch inst.opcode {
				case opPrint:

					// Print called between Spin and Stop
					// demands special consideration.
					if spinner != "" {
						if isTerminal {
							fmt.Fprint(stderr, "\r", s, " ", dots, ". (to be continued)\n")
						} else {
							fmt.Fprintln(stderr, " (to be continued)")
						}
						dots, s = "", "(continuing)"
					}

					fmt.Fprintln(stderr, inst.s) // TODO split on word boundaries to make long messages easy to read on narrow terminals

					// Per above, indicate that the spinning is resuming.
					if spinner != "" {
						fmt.Fprint(stderr, "(continuing)")
					}

				case opQuiet:
					stderr, err = os.Open(os.DevNull)
					if err != nil {
						log.Fatal(err)
					}

				case opSpin:

					// The last line of output on the terminal can't wrap or
					// carriage returns will make a mess of things.
					var i int
					if isTerminal {
						i = len(inst.s) - len(inst.s)%width
						if i > 0 {
							fmt.Fprintln(stderr, inst.s[:i])
						}
					}
					dots, s, spinner = "", inst.s[i:], "-"
					fmt.Fprint(stderr, s, " ", dots, spinner)

				case opStop:

					// No carriage returns if standard output is not a terminal.
					if !isTerminal {
						fmt.Fprint(stderr, " ", strings.TrimSuffix(inst.s, "\n"), "\n")
						break
					}

					fmt.Fprint(stderr, "\r", s, " ", dots, ". ", strings.TrimSuffix(inst.s, "\n"), "\n")
					dots, s, spinner = "", "", ""
				}
				inst.ch <- struct{}{}

			case <-tick:

				// No carriage returns if standard output is not a terminal.
				if !isTerminal {
					continue
				}

				if ticks%(2*hz) == 0 {
					dots += "."
				}
				if spinner != "" {

					// If the spinner is about to wrap, output a newline and
					// align it to continue below.
					if len(fmt.Sprint("\r", s, " ", dots)) > width {
						fmt.Fprint(stderr, "\r", s, " ", dots, "\n")
						dots, s = "", strings.Repeat(" ", len(s))
					}

					fmt.Fprint(stderr, "\r", s, " ", dots, spinner)
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

func op(opcode int, s string) {
	ch := make(chan struct{})
	chInst <- instruction{
		ch:     ch,
		opcode: opcode,
		s:      s,
	}
	<-ch
}
