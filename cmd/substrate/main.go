package main

import (
	"context"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/cmd/substrate/account"
	"github.com/src-bin/substrate/cmd/substrate/accounts"
	assumerole "github.com/src-bin/substrate/cmd/substrate/assume-role"
	"github.com/src-bin/substrate/cmd/substrate/credentials"
	deletestaticaccesskeys "github.com/src-bin/substrate/cmd/substrate/delete-static-access-keys"
	intranetzip "github.com/src-bin/substrate/cmd/substrate/intranet-zip"
	"github.com/src-bin/substrate/cmd/substrate/upgrade"
	"github.com/src-bin/substrate/cmd/substrate/whoami"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

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
		CompletionOptions:     cobra.CompletionOptions{DisableDefaultCmd: true},
		DisableFlagsInUseLine: true,
	}
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
		DisableFlagsInUseLine: true,
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "TODO versionCmd.Short",
		Long:  `TODO versionCmd.Long`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			version.Print()
		},
		DisableFlagsInUseLine: true,
	})

	rootCmd.AddCommand(account.Command())
	rootCmd.AddCommand(assumerole.Command())
	rootCmd.AddCommand(credentials.Command())
	rootCmd.AddCommand(deletestaticaccesskeys.Command())
	rootCmd.AddCommand(intranetzip.Command())
	rootCmd.AddCommand(upgrade.Command())
	rootCmd.AddCommand(whoami.Command())

	// Breadcrumbs to deprecated subcommands.
	rootCmd.AddCommand(accounts.Command())

	rootCmd.ExecuteContext(ctx)
}
