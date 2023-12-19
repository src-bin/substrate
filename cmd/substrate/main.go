package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

//go:generate go run ../../tools/dispatch-map/main.go -function Main -o dispatch-map-Main.go .
//go:generate go run ../../tools/dispatch-map/main.go -function Synopsis -o dispatch-map-Synopsis.go .

func main() {

	// XXX EXPERIMENTAL COBRA IMPLEMENTATION OF SUBSTRATE XXX
	run := func(cmd *cobra.Command, args []string) {
		log.Printf("Run %s cmd.Flags(): %+v, args: %+v)", cmd.Use, cmd.Flags(), args)
	}
	var shellCompletionFlag, versionFlag bool
	shellCompletionCmd := &cobra.Command{
		Use:    "shell-completion",
		Hidden: true,
		Short:  "TODO",
		Long:   `TODO`,
		Run: func(cmd *cobra.Command, args []string) {
			shell := os.Getenv("SHELL")
			if shell != "" {
				shell = filepath.Base(shell)
			}
			switch shell {
			case "", "bash":
				cmd.Root().GenBashCompletionV2(os.Stdout, true /* includeDesc */)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true /* includeDesc */)
			default:
				ui.Fatalf("unsupported SHELL=%q", shell)
			}
		},
	}
	rootCmd := &cobra.Command{
		Use:   "substrate",
		Short: "TODO",
		Long:  `TODO`,
		Run: func(cmd *cobra.Command, args []string) {
			if shellCompletionFlag {
				shellCompletionCmd.Run(cmd, args)
			} else if versionFlag {
				version.Print()
				return
			} else {
				run(cmd, args)
			}
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Flags().BoolVar(&shellCompletionFlag, "shell-completion", false, "TODO")
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "TODO")
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "TODO",
		Long:  `TODO`,
	}
	accountCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "TODO",
		Long:  `TODO`,
		Run:   run,
	})
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(shellCompletionCmd)
	ui.Must(rootCmd.Execute())
	return
	// XXX EXPERIMENTAL COBRA IMPLEMENTATION OF SUBSTRATE XXX

	if version.IsTrial() {
		ui.Print("this is a trial version of Substrate; contact <sales@src-bin.com> for support and the latest version")
	}

	// Dispatch to the package named like the subcommand with a Main function.
	m, ok := DispatchMapMain.Map[os.Args[1]]
	if !ok {
		ui.Fatal(ok)
	}
	subcommand := os.Args[1]
	os.Args = append([]string{fmt.Sprintf("%s-%s", os.Args[0], os.Args[1])}, os.Args[2:]...) // so m.Func can flag.Parse()
	for m.Func == nil {
		m, ok = m.Map[os.Args[1]]
		if !ok {
			ui.Fatal(ok)
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
