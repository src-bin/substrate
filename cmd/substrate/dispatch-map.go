package main

var dispatchMap = map[string]func(includeInDispatchMap){
	"whoami": whoami,
}
