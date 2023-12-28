package deletestaticaccesskeys

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/versionutil"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:    "delete-static-access-keys",
		Hidden: true,
		Short:  "delete the Substrate IAM user's static access keys",
		Long:   ``,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
	}
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {

	cfg = awscfg.Must(cfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	versionutil.PreventDowngrade(ctx, cfg)

	cfg.Telemetry().FinalAccountId = aws.ToString(cfg.MustGetCallerIdentity(ctx).Account)
	cfg.Telemetry().FinalRoleName = roles.OrganizationAdministrator

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	ui.Spin("deleting all access keys for the Substrate user")
	if err := awsiam.DeleteAllAccessKeys(ctx, cfg, users.Substrate, 0); err != nil {
		log.Fatal(err)
	}
	ui.Stop("done")

	_, err := ui.Prompt("visit <https://console.aws.amazon.com/iam/home#/security_credentials> and delete all root access keys (which can't be deleted via the API) and press <enter>")
	if err != nil {
		log.Fatal(err)
	}

	ui.Printf(
		"from now on, use `eval $(substrate credentials)` or <https://%s/credential-factory> to mint short-lived AWS access keys",
		naming.MustIntranetDNSDomainName(),
	)

}
