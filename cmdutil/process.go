package cmdutil

import (
	"fmt"
	"os"
)

// ProcessName returns the process name of the process with the given pid.
func ProcessName(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid pid: %d", pid)
	}

	return processName(pid)
}

// ParentProcessName returns the name of the parent process of the current process.
func ParentProcessName() (string, error) {
	return ProcessName(os.Getppid())
}
