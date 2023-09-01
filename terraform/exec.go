package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/src-bin/substrate/ui"
)

func Apply(dirname string, autoApprove bool) error {
	ui.Printf("applying Terraform changes in %s", dirname)
	//log.Print(execdlp(dirname, "aws", "sts", "get-caller-identity"))
	if autoApprove {
		return execdlp(dirname, "terraform", "apply", "-auto-approve")
	}
	return execdlp(dirname, "terraform", "apply")
}

func Destroy(dirname string, autoApprove bool) error {
	ui.Printf("destroying Terraform-managed resources in %s", dirname)
	if autoApprove {
		return execdlp(dirname, "terraform", "destroy", "-auto-approve")
	}
	return execdlp(dirname, "terraform", "destroy")
}

func Fmt(dirname string) error {
	ui.Printf("formatting Terraform source files in %s", dirname)
	return execdlp(dirname, "terraform", "fmt")
}

func Init(dirname string) error {
	ui.Printf("initializing Terraform in %s", dirname)
	return execdlp(dirname, "terraform", "init", "-reconfigure", "-upgrade")
}

func InstalledVersion() (string, error) {
	if memoizedVersion != "" {
		return memoizedVersion, nil
	}
	cmd := exec.Command("terraform", "version", "-json")
	cmd.Stdin = os.Stdin
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	b := stdout.Bytes()

	if len(b) > 1 && b[0] == 'T' { // "Terraform v0.12."x
		if _, err := fmt.Sscanf(string(b), "Terraform v%s\n", &memoizedVersion); err != nil {
			return "", err
		}
		return memoizedVersion, nil
	}

	out := struct {
		Version string `json:"terraform_version"`
	}{}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	memoizedVersion = out.Version

	return memoizedVersion, nil
}

func Plan(dirname string) error {
	ui.Printf("planning Terraform changes in %s", dirname)
	//log.Print(execdlp(dirname, "aws", "sts", "get-caller-identity"))
	err := execdlp(dirname, "terraform", "plan")
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return err
	}
	if exitErr.ExitCode() != 0 {
		return nil // it's OK if a plan fails; that's useful data
	}
	return err
}

func ShortInstalledVersion() (string, error) {
	version, err := InstalledVersion()
	if err != nil {
		return "", err
	}
	parts := strings.SplitN(version, ".", 3)
	return strings.Join(parts[:2], "."), nil
}

func Upgrade(dirname string) error {
	shortVersion, err := ShortInstalledVersion()
	if err != nil {
		return err
	}

	// Substrate started in the era of Terraform 0.12 and, coincidentally, its
	// upgrade program is not idempotent. Let's skip that whole sad party.
	if shortVersion == "0.12" {
		return nil
	}

	ui.Printf("upgrading Terraform module in %s to Terraform version %s", dirname, shortVersion)
	return execdlp(dirname, "terraform", fmt.Sprintf("%supgrade", shortVersion), "-yes")
}

// execdlp executes progname in dirname (or, implicitly the current working
// directory if dirname is empty) with args as its arguments and all the
// standard I/O file descriptors inherited from the forking process.
func execdlp(dirname, progname string, args ...string) error {
	cmd := exec.Command(progname, args...)
	cmd.Dir = dirname
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// execlp executes progname in the current working directory with args as its
// arguments and all the standard I/O file descriptors inherited from the
// forking process.
func execlp(progname string, args ...string) error {
	return execdlp("", progname, args...)
}

var memoizedVersion string
