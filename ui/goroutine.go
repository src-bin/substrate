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

					fmt.Println(inst.s) // TODO split on word boundaries to make long messages easy to read on narrow terminals

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
