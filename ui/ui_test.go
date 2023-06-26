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
