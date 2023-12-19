package main

import (
	"context"
	"fmt"
	"os"
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

	if version.IsTrial() {
		ui.Print("this is a trial version of Substrate; contact <sales@src-bin.com> for support and the latest version")
	}

	// Dispatch to the package named like the subcommand with a Main function.
	m, ok := DispatchMapMain.Map[os.Args[1]]
	if !ok {
		usage(1)
	}
	subcommand := os.Args[1]
	os.Args = append([]string{fmt.Sprintf("%s-%s", os.Args[0], os.Args[1])}, os.Args[2:]...) // so m.Func can flag.Parse()
	for m.Func == nil {
		m, ok = m.Map[os.Args[1]]
		if !ok {
			usage(1)
		}
		subcommand = fmt.Sprintf("%s %s", subcommand, os.Args[1])
		os.Args = append([]string{fmt.Sprintf("%s-%s", os.Args[0], os.Args[1])}, os.Args[2:]...) // so m.Func can flag.Parse()
	}
	ui.Must(cmdutil.Chdir())
	u, err := user.Current()
	ui.Must(err)
	ctx := contextutil.WithValues(context.Background(), "substrate", subcommand, u.Username)
	cfg, err := awscfg.NewConfig(ctx) // TODO takes 0.8s!
	ui.Must(err)
	m.Func(ctx, cfg, os.Stdout)

	// If no one's posted telemetry yet, post it now, and wait for it to finish.
	cfg.Telemetry().Post(ctx)
	cfg.Telemetry().Wait(ctx)

}

func usage(status int) {
	var commands []string

	for subcommand, _ := range DispatchMapMain.Map { // TODO hand-write this message; this auto-generated one is trash
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
	ui.Print("if you're unsure where to start, visit <https://docs.substrate.tools/>")

	os.Exit(status)
}
