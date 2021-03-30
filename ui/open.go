package ui

import (
	"os/exec"
)

// OpenURL opens u in a browser (or other appropriate app depending on the
// scheme and local configuration) and returns true or falls back to printing
// the URL with instructions to open it and returns false.
func OpenURL(s string) bool {

	// This part could be safely memoized if it ever became the slow part.
	var progname string
	for _, progname = range []string{
		"open",     // MacOS
		"xdg-open", // Linux
	} {
		if _, err := exec.LookPath(progname); err == nil {
			break
		}
	}

	if progname != "" {
		Printf("opening <%s> in your web browser", s)
		if err := exec.Command(progname, s).Start(); err != nil {
			Fatal(err)
		}
		return true
	} else {
		Printf("please open <%s> in your web browser", s)
		return false
	}
}
