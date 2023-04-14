package main

import (
	"time"

	"github.com/src-bin/substrate/ui"
)

func main() {
	ui.Spin("foo")
	time.Sleep(5 * time.Second)
	ui.Print("interrupting")
	time.Sleep(5 * time.Second)
	ui.Spin("bar")
	time.Sleep(5 * time.Second)
	ui.Spin("baz")
	time.Sleep(5 * time.Second)
	ui.Print("interrupting")
	time.Sleep(5 * time.Second)
	ui.Stop("ok")
	time.Sleep(5 * time.Second)
	ui.Stop("ok")
	time.Sleep(5 * time.Second)
	ui.Stop("ok")
	ui.Print("done")
}
