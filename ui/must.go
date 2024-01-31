package ui

import "fmt"

func Must(err error) {
	if err != nil {
		op(opFatal, fmt.Sprint(withCaller(helpful(err))...))
	}
}

func Must2[T any](v T, err error) T {
	if err != nil {
		op(opFatal, fmt.Sprint(withCaller(helpful(err))...))
	}
	return v
}
