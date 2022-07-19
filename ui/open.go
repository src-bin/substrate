package ui

import (
	"os/exec"
)

// OpenURL opens u in a browser (or other appropriate app depending on the
// scheme and local configuration) and returns true or falls back to printing
// the URL with instructions to open it and returns false.
func OpenURL(s string) bool {
	if progname := lookPath(); progname != "" {
		Printf("opening <%s> in your web browser", s)
		if err := exec.Command(progname, s).Start(); err != nil {
			Fatal(err)
		}
		return true
	}
	Printf("please open <%s> in your web browser", s)
	return false
}

// lookPath returns the program that can be used to generically open a URL in
// the default web browser or the empty string if there isn't one. It could be
// safely memoized if it ever became the slow part.
func lookPath() string {
	for _, progname := range []string{
		"open",     // MacOS
		"xdg-open", // Linux
	} {
		if _, err := exec.LookPath(progname); err == nil {
			return progname
		}
	}
	return ""
}
