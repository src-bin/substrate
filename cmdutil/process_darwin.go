//go:build darwin
// +build darwin

package cmdutil

import (
	"golang.org/x/sys/unix"
)

// processNameDarwin returns the process name of the process with the given pid on macOS.
func processName(pid int) (string, error) {
	kinfoProc, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return "", err
	}

	nameBytes := kinfoProc.Proc.P_comm[:]
	for i, b := range nameBytes {
		if b == 0 {
			nameBytes = nameBytes[:i]
			break
		}
	}
	return string(nameBytes), nil
}
