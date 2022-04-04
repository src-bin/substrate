package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

//go:generate go run ../../tools/dispatch-map/main.go -package main .

func main() {

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
	if err != nil {
		ui.Fatal(err)
	}
	if executable, err = filepath.EvalSymlinks(executable); err != nil {
		ui.Fatal(err)
	}
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
		case "-version", "--version":
			version.Print()
			os.Exit(0)

		}
		os.Args = append([]string{fmt.Sprintf("substrate-%s", os.Args[1])}, os.Args[2:]...)
	}

	// Dispatch to the package named like the subcommand with a Main function
	// or to the appropriate substrate-* program.
	subcommand := strings.TrimPrefix(filepath.Base(os.Args[0]), "substrate-")
	f, ok := dispatchMap[subcommand]
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

	ctx := context.WithValue(
		context.WithValue(
			context.Background(),
			"Command",
			"substrate",
		),
		"Subcommand",
		subcommand,
	)
	cfg, err := awscfg.NewMain(ctx)
	if err != nil {
		ui.Fatal(err)
	}
	f(ctx, cfg)

	// If no one's posted telemetry yet, post it now, and wait for it to finish.
	cfg.Telemetry().Post(ctx)
	cfg.Telemetry().Wait(ctx)

}

func shellCompletion() {
	defer os.Exit(0)

	// The argument structure bash(1) uses with `complete -C` appears to
	// follow typical calling convention with argv[0], then give the most
	// recently typed argument, and then the previously typed argument as
	// some kind of confusing convenience. If the command needs the entire
	// typed command, it's available in the COMP_LINE environment variable.
	word := os.Args[3]
	//log.Printf("word: %q", word)
	previousWord := os.Args[4]
	//log.Printf("previousWord: %q", previousWord)

	if previousWord == "substrate" {
		if _, ok := dispatchMap[word]; ok {
			fmt.Println(word)
			return
		}
		var subcommands []string
		for subcommand, _ := range dispatchMap {
			if strings.HasPrefix(subcommand, word) {
				subcommands = append(subcommands, subcommand)
				//log.Printf("prefix match: %q for %q", word, subcommand)
			}
		}
		sort.Strings(subcommands)
		for _, subcommand := range subcommands {
			fmt.Println(subcommand)
		}
		return
	}

}

func usage(status int) {
	var commands []string

	for subcommand, _ := range dispatchMap {
		commands = append(commands, fmt.Sprintf("substrate-%s", subcommand))
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
		if name := entry.Name(); strings.HasPrefix(name, "substrate-") {
			commands = append(commands, name)
		}
	}

	ui.Print("Substrate is an opinionated suite of tools that manage secure, reliable, and compliant cloud infrastructure in AWS")
	ui.Print("the following commands are available:")
	sort.Strings(commands)
	var previousCommand string
	for _, command := range commands {
		if command != previousCommand {
			ui.Printf("\t%s", command)
		}
		previousCommand = command
	}
	ui.Print("if you're unsure where to start, visit <https://src-bin.com/substrate/manual/>")

	os.Exit(status)
}
