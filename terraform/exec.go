package terraform

import (
	"os"
	"os/exec"

	"github.com/src-bin/substrate/ui"
)

func Apply(dirname string) error {
	ui.Printf("applying Terraform changes in %s", dirname)
	cmd := exec.Command("make", "-C", dirname, "apply")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Fmt() error {
	ui.Spin("formatting Terraform source files")
	defer ui.Stop("done")
	cmd := exec.Command("terraform", "fmt", "-recursive")
	return cmd.Run()
}

func Init(dirname string) error {
	ui.Printf("initializing Terraform in %s", dirname)
	cmd := exec.Command("make", "-C", dirname, "init")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
