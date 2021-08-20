package main

import (
	"github.com/src-bin/substrate/cmd/substrate/whoami"
)

var dispatchMap = map[string]func(){
	"whoami": whoami.Main,
}
