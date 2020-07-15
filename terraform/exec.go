package terraform

import (
	"os"
	"os/exec"

	"github.com/src-bin/substrate/ui"
)

func Apply(dirname string, autoApprove bool) error {
	ui.Printf("applying Terraform changes in %s", dirname)
	if autoApprove {
		return execlp("make", "-C", dirname, "apply", "AUTO_APPROVE=-auto-approve")
	}
	return execlp("make", "-C", dirname, "apply")
}

func Destroy(dirname string, autoApprove bool) error {
	ui.Printf("destroying Terraform-managed resources in %s", dirname)
	if autoApprove {
		return execlp("make", "-C", dirname, "destroy", "AUTO_APPROVE=-auto-approve")
	}
	return execlp("make", "-C", dirname, "destroy")
}

func Fmt(dirname string) error {
	ui.Print("formatting Terraform source files")
	return execlp("terraform", "fmt", dirname)
}

func Init(dirname string) error {
	ui.Printf("initializing Terraform in %s", dirname)
	return execlp("make", "-C", dirname, "init")
}

func Plan(dirname string) error {
	ui.Printf("planning Terraform changes in %s", dirname)
	return execlp("make", "-C", dirname, "plan")
}

func execlp(progname string, args ...string) error {
	cmd := exec.Command(progname, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
