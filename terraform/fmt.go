package terraform

import "os/exec"

func Fmt() error {
	cmd := exec.Command("terraform", "fmt", "-recursive")
	return cmd.Run()
}
