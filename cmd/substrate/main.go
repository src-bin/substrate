package main

import (
	"context"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/cmd/substrate/account"
	"github.com/src-bin/substrate/cmd/substrate/credentials"
	"github.com/src-bin/substrate/cmd/substrate/whoami"
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

	ui.Must(cmdutil.Chdir())
	u, err := user.Current()
	ui.Must(err)
	var command, subcommand string
	if len(os.Args) >= 1 {
		command = os.Args[0]
	}
	if len(os.Args) >= 2 && len(os.Args[1]) >= 1 && os.Args[1][0] != '-' {
		subcommand = os.Args[1] // only valid because there are no flags before subcommands
	}
	if len(os.Args) >= 3 && len(os.Args[2]) >= 1 && os.Args[2][0] != '-' {
		subcommand += " " + os.Args[2] // same as above
	}
	ctx := contextutil.WithValues(
		context.Background(),
		command,
		subcommand,
		u.Username,
	)

	var versionFlag bool
	rootCmd := &cobra.Command{
		Use:   "substrate",
		Short: "TODO rootCmd.Short",
		Long:  `TODO rootCmd.Long`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if versionFlag {
				version.Print()
			} else {
				cmd.Help()
			}
		},
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "TODO versionFlag")
	rootCmd.AddCommand(&cobra.Command{
		Use:    "shell-completion",
		Hidden: true,
		Short:  "TODO shellCompletionCmd.Short",
		Long:   `TODO shellCompletionCmd.Long`,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			var shell string
			if cmdutil.CheckForFish() {
				shell = "fish"
			} else {
				shell = os.Getenv("SHELL")
			}
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
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "TODO versionCmd.Short",
		Long:  `TODO versionCmd.Long`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			version.Print()
		},
	})

	rootCmd.AddCommand(account.Command())

	rootCmd.AddCommand(credentials.Command())

	rootCmd.AddCommand(whoami.Command())

	log.Print(time.Now())
	rootCmd.ExecuteContext(ctx)
	log.Print(time.Now())

	// If no one's posted telemetry yet, post it now, and wait for it to finish.
	/* XXX SLOW AS FUCK
		cfg.Telemetry().Post(ctx)
		cfg.Telemetry().Wait(ctx)
	XXX */

}
