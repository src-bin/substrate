package ui

import (
	"testing"
	"time"
)

func ExampleUI() {
	Print("hi")
	Spin("spinnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnnn")
	time.Sleep(13e9)
	Print("mooooooo")
	time.Sleep(5e9)
	Stop("bye")
}

func TestPrintfWithCaller(t *testing.T) {
	PrintfWithCaller("this is a visual test which is bad but is better than no test")
}

func TestSpinPrintStop(t *testing.T) {
	t.Skip()
	Spin("testing")
	time.Sleep(5e9)
	Spin("nesting")
	time.Sleep(5e9)
	Print("interrupting the spinner")
	time.Sleep(5e9)
	Print("and again")
	time.Sleep(5e9)
	Stop("done")
	time.Sleep(5e9)
	Stop("really done")

	// Ouput if stderr is a TTY and thus the spinners spin:
	/*
	   testing ...
	    nesting ...
	     interrupting the spinner
	    ....
	     and again
	    .... done
	   ... really done
	*/

	// Output if stderr is not a TTY:
	/*
	   testing ...
	    nesting ...
	     interrupting the spinner
	     and again
	    ... done
	   ... really done
	*/

}
