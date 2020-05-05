package terraform

import "os/exec"

func Apply() error {
	cmd := exec.Command("terraform", "apply")
	return cmd.Run()
}
