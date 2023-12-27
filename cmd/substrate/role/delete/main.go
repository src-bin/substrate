package delete

import (
	"context"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

var (
	roleName = new(string)
	force    = new(bool)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete --role <role> [--force] [--quiet]",
		Short: "TODO create.Command().Short",
		Long:  `TODO create.Command().Long`,
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--role",
				"--force",
				"--quiet",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().StringVar(roleName, "role", "", "name of the IAM role to create")
	cmd.RegisterFlagCompletionFunc("role", cmdutil.NoCompletionFunc)
	cmd.Flags().BoolVar(force, "force", false, "delete the role from all accounts without confirmation")
	cmd.Flags().AddFlag(cmdutil.QuietFlag())
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {
	if *roleName == "" {
		ui.Fatal(`-role "..." is required`)
	}
	if *roleName == roles.Administrator || *roleName == roles.Auditor {
		ui.Fatalf("cannot delete %s roles with `substrate role delete`", *roleName)
	}

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	versionutil.PreventDowngrade(ctx, cfg)

	allAccounts, err := awsorgs.ListAccounts(ctx, cfg)
	ui.Must(err)
	var found bool
	for _, account := range allAccounts {
		if account.Tags[tagging.SubstrateSpecialAccount] == naming.Audit || account.Tags[tagging.SubstrateType] == naming.Audit {
			continue
		}
		if err := awsiam.DeleteRoleWithConfirmation(
			ctx,
			awscfg.Must(account.Config(
				ctx,
				cfg,
				account.AdministratorRoleName(),
				time.Hour,
			)),
			*roleName,
			*force,
		); err == nil {
			found = true
		} else if !awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
			ui.Fatal(err)
		}
	}

	// Print a warning if we did not delete _any_ roles as this might mean
	// the user misspelled the role's name.
	if !found {
		ui.Printf("did not find any roles named %q", roleName)
	}

}
