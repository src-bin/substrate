package deleterole

import (
	"context"
	"flag"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	del := flag.Bool("delete", false, "delete the role from all accounts without confirmation")
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

	allAccounts, err := awsorgs.ListAccounts(ctx, cfg)
	ui.Must(err)
	for _, a := range allAccounts {
		if a.Tags[tagging.SubstrateSpecialAccount] == naming.Audit {
			continue
		}
		assumedCfg := awscfg.Must(cfg.AssumeRole(
			ctx,
			aws.ToString(a.Id),
			a.AdministratorRoleName(),
			time.Hour,
		))

		if !*del {
			_, err := awsiam.GetRole(
				ctx,
				assumedCfg,
				*roleName,
			)
			if awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
				continue
			}
			ui.Must(err)
			ok, err := ui.Confirmf("delete role %s in %s? (yes/no)", *roleName, a)
			ui.Must(err)
			if !ok {
				continue
			}
		}

		err = awsiam.DeleteRolePolicy(ctx, assumedCfg, *roleName)
		if awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
			err = nil
		}
		ui.Must(err)
		err = awsiam.DeleteRole(ctx, assumedCfg, *roleName)
		if awsutil.ErrorCodeIs(err, awsiam.NoSuchEntity) {
			err = nil
		}
		ui.Must(err)
	}
}
