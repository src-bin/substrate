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
	createaccount "github.com/src-bin/substrate/cmd/substrate/create-account"
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	"github.com/src-bin/substrate/cmd/substrate/credentials"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	intranetzip "github.com/src-bin/substrate/cmd/substrate/intranet-zip"
	"github.com/src-bin/substrate/cmd/substrate/role"
	"github.com/src-bin/substrate/cmd/substrate/roles"
	"github.com/src-bin/substrate/cmd/substrate/setup"
	"github.com/src-bin/substrate/cmd/substrate/terraform"
	"github.com/src-bin/substrate/cmd/substrate/upgrade"
	"github.com/src-bin/substrate/cmd/substrate/whoami"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/features"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func main() {

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

	if features.MacOSKeychain.Enabled() {
		ui.Must(cmdutil.SetenvFromTPM())
	}

	var versionFlag bool
	rootCmd := &cobra.Command{
		Use:   "substrate",
		Short: "Substrate: the right way to AWS",
		Long: `Substrate: the right way to AWS

<https://docs.substrate.tools/>`,
		Args:                  cobra.NoArgs,
		CompletionOptions:     cobra.CompletionOptions{DisableDefaultCmd: true},
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			if versionFlag {
				version.Print()
			} else {
				cmd.Help()
			}
		},
	}
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "print Substrate version and exit")
	rootCmd.AddCommand(&cobra.Command{
		Use:    "shell-completion",
		Hidden: true,
		Short:  "print shell completion program for the current shell",
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
		Short: "print Substrate version and exit",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			version.Print()
		},
		DisableFlagsInUseLine: true,
	})

	rootCmd.AddCommand(account.Command())
	rootCmd.AddCommand(assumerole.Command())
	rootCmd.AddCommand(credentials.Command())
	rootCmd.AddCommand(intranetzip.Command())
	rootCmd.AddCommand(role.Command())
	rootCmd.AddCommand(setup.Command())
	rootCmd.AddCommand(terraform.Command())
	rootCmd.AddCommand(upgrade.Command())
	rootCmd.AddCommand(whoami.Command())

	// Breadcrumbs to deprecated subcommands.
	rootCmd.AddCommand(accounts.Command())
	rootCmd.AddCommand(createaccount.Command())
	rootCmd.AddCommand(createrole.Command())
	rootCmd.AddCommand(deleterole.Command())
	rootCmd.AddCommand(roles.Command())

	rootCmd.ExecuteContext(ctx) // TODO RunE; let Main return an error; avoid os.Exit(1) breaking defer; use panic/recover in ui.Fatal
}
