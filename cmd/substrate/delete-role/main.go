package deleterole

import (
	"context"
	"flag"
	"io"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	force := flag.Bool("delete", false, "delete the role from all accounts without confirmation")
	roleName := flag.String("role", "", "name of the IAM role to delete")
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	flag.Usage = func() {
		ui.Print("Usage: substrate delete-role [-delete] -role <role> [-quiet]")
	}
	flag.Parse()
	version.Flag()
	if *quiet {
		ui.Quiet()
	}
	if *roleName == "" {
		ui.Fatal(`-role "..." is required`)
	}

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	versionutil.PreventDowngrade(ctx, cfg)

	allAccounts, err := awsorgs.ListAccounts(ctx, cfg)
	ui.Must(err)
	for _, account := range allAccounts {
		if account.Tags[tagging.SubstrateSpecialAccount] == naming.Audit {
			continue
		}
		ui.Must(awsiam.DeleteRoleWithConfirmation(
			ctx,
			awscfg.Must(account.Config(
				ctx,
				cfg,
				account.AdministratorRoleName(),
				time.Hour,
			)),
			*roleName,
			*force,
		))
	}
}
