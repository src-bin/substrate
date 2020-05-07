package terraform

import "os/exec"

func Apply() error {
	cmd := exec.Command("terraform", "apply")
	return cmd.Run()
}

// TODO maybe also generate a Makefile for each Terraform entrypoint with targets like plan and apply?
