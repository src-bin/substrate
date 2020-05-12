package terraform

import (
	"os"
	"os/exec"

	"github.com/src-bin/substrate/ui"
)

func Apply(dirname string) error {
	ui.Printf("applying Terraform changes in %s", dirname)
	return execlp("make", "-C", dirname, "apply")
}

func Fmt() error {
	ui.Spin("formatting Terraform source files")
	defer ui.Stop("done")
	return execlp("terraform", "fmt", "-recursive")
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
