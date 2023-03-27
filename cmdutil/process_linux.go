//go:build linux
// +build linux

package cmdutil

import (
	"fmt"
	"os"
	"strings"
)

// processNameLinux returns the process name of the process with the given pid on Linux.
func processName(pid int) (string, error) {
	commFile := fmt.Sprintf("/proc/%d/comm", pid)
	contents, err := os.ReadFile(commFile)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
