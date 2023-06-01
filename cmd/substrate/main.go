package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

//go:generate go run ../../tools/dispatch-map/main.go -function Main -o dispatch-map-Main.go .
//go:generate go run ../../tools/dispatch-map/main.go -function Synopsis -o dispatch-map-Synopsis.go .

func main() {

	if version.IsTrial() {
		ui.Print("this is a trial version of Substrate; contact <sales@src-bin.com> for support and the latest version")
	}

	// If we were invoked directly, expect to find a subcommand in the first
	// position. Reconfigure the arguments to make it look like we were invoked
	// via a symbolic link so as to unify dispatch and simply argument parsing
	// after dispatch.
	//
	// The extra call to filepath.EvalSymlinks is to normalize executable to
	// be a reference to the actual binary. On MacOS, os.Executable returns
	// the pathname of the symbolic link, which caused the comparison with
	// os.Args[0] to always succeed.
	executable, err := os.Executable()
	ui.Must(err)
	executable, err = filepath.EvalSymlinks(executable)
	ui.Must(err)
	if filepath.Base(os.Args[0]) == filepath.Base(executable) {
		if len(os.Args) < 2 {
			usage(1)
		}
		switch os.Args[1] {

		// Respond to `substrate -h` but not `substrate-* -h` or
		// `substrate * -h`, which are handled by main.main or *.Main.
		case "-h", "-help", "--help":
			usage(0)

		// Dispatch shell completion from here so we can get in and out before
		// we go into all the awscfg business, which is too slow to do
		// keystroke-by-keystroke.
		case "-shell-completion", "--shell-completion", "-shell-completion=bash", "--shell-completion=bash":
			shellCompletion()

		// Respond to -version or --version, however folks want to call it.
		case "-v", "-version", "--version":
			version.Print()
			os.Exit(0)

		}
		os.Args = append([]string{fmt.Sprintf("substrate-%s", os.Args[1])}, os.Args[2:]...)
	}

	// Dispatch to the package named like the subcommand with a Main function
	// or to the appropriate substrate-* program.
	subcommand := strings.TrimPrefix(filepath.Base(os.Args[0]), "substrate-")
	f, ok := dispatchMapMain[subcommand]
	if !ok {
		ui.Printf("dispatching %s, which is deprecated", os.Args[0])
		if _, err := exec.LookPath(os.Args[0]); err != nil {
			ui.Fatalf("%s not found", os.Args[0])
		}
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1)
		}
		os.Exit(0)
	}

	ui.Must(cmdutil.Chdir())
	u, err := user.Current()
	ui.Must(err)
	ctx := contextutil.WithValues(context.Background(), "substrate", subcommand, u.Username)
	cfg, err := awscfg.NewConfig(ctx)
	ui.Must(err)
	f(ctx, cfg, os.Stdout)

	// If no one's posted telemetry yet, post it now, and wait for it to finish.
	cfg.Telemetry().Post(ctx)
	cfg.Telemetry().Wait(ctx)

}

func usage(status int) {
	var commands []string

	for subcommand, _ := range dispatchMapMain {
		commands = append(commands, fmt.Sprintf("substrate %s", subcommand))
	}

	executable, err := os.Executable()
	if err != nil {
		ui.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(executable))
	if err != nil {
		ui.Fatal(err)
	}
	for _, entry := range entries {
		if name := entry.Name(); strings.HasPrefix(name, "substrate-") && entry.Type() != os.ModeSymlink {
			commands = append(commands, name)
		}
	}

	ui.Print("Substrate manages secure, reliable, and compliant cloud infrastructure in AWS")
	ui.Print("the following commands are available:")
	sort.Strings(commands)
	var previousCommand string
	for _, command := range commands {
		if command != previousCommand {
			ui.Printf("\t%s", command)
		}
		previousCommand = command
	}
	ui.Print("if you're unsure where to start, visit <https://docs.src-bin.com/substrate/>")

	os.Exit(status)
}
