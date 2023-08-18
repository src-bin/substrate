package ui

import "encoding/json"

func Debug(args ...interface{}) {
	for i, arg := range args {
		b, err := json.MarshalIndent(arg, "", "\t")
		Must(err)
		if i > 0 {
			args[i] = ", " + string(b)
		} else {
			args[i] = string(b)
		}
	}
	Print(withCaller(args...)...)
}
