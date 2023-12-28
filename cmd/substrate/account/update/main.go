package update

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/versionutil"
)

var (
	domain, domainFlag, domainCompletionFunc                = cmdutil.DomainFlag("domain of the AWS account to update")
	environment, environmentFlag, environmentCompletionFunc = cmdutil.EnvironmentFlag("environment of the AWS account to update")
	quality, qualityFlag, qualityCompletionFunc             = cmdutil.QualityFlag("quality of the AWS account to update")
	autoApprove, noApply                                    = new(bool), new(bool)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update --domain <domain> --environment <environment> [--quality <quality>] [--auto-approve|--no-apply]",
		Short: "update an existing AWS account and plan or apply its root Terraform modules",
		Long:  ``,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--domain", "--environment", "--quality",
				"--auto-approve", "--no-apply",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().AddFlag(domainFlag)
	cmd.RegisterFlagCompletionFunc(domainFlag.Name, domainCompletionFunc)
	cmd.Flags().AddFlag(environmentFlag)
	cmd.RegisterFlagCompletionFunc(environmentFlag.Name, environmentCompletionFunc)
	cmd.Flags().AddFlag(qualityFlag)
	cmd.RegisterFlagCompletionFunc(qualityFlag.Name, qualityCompletionFunc)
	cmd.Flags().BoolVar(autoApprove, "auto-approve", false, "apply Terraform changes without waiting for confirmation")
	cmd.Flags().BoolVar(noApply, "no-apply", false, "do not apply Terraform changes")
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {
	if *environment != "" && *quality == "" {
		*quality = cmdutil.QualityForEnvironment(*environment)
	}
	if *domain == "" || *environment == "" || *quality == "" {
		ui.Fatal(`--domain "..." --environment "..." --quality"..." are required`)
	}
	if d := *domain; d == "admin" || d == "common" || d == "deploy" || d == "intranet" || d == "lambda-function" || d == "network" || d == "peering-connection" || d == "substrate" {
		ui.Fatalf("--domain %q is reserved; please choose a different name", d)
	}
	if strings.ContainsAny(*domain, ",. ") {
		ui.Fatalf("--domain %q cannot contain commas or spaces", *domain)
	}
	if strings.ContainsAny(*environment, ", ") {
		ui.Fatalf("--environment %q cannot contain commas or spaces", *environment)
	}
	if strings.ContainsAny(*quality, ", ") {
		ui.Fatalf("--quality %q cannot contain commas or spaces", *quality)
	}
	veqpDoc, err := veqp.ReadDocument()
	ui.Must(err)
	if !veqpDoc.Valid(*environment, *quality) {
		ui.Fatalf("--environment %q --quality %q is not a valid environment and quality pair in your organization", *environment, *quality)
	}

	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	versionutil.PreventDowngrade(ctx, mgmtCfg)

	ui.Spin("finding the account")
	account, err := mgmtCfg.FindServiceAccount(ctx, *domain, *environment, *quality)
	ui.Must(err)
	if account == nil {
		ui.Stop("not found")
		os.Exit(1)
	}
	ui.Stop(account)

	mgmtCfg.Telemetry().FinalAccountId = aws.ToString(account.Id)
	mgmtCfg.Telemetry().FinalRoleName = roles.Administrator

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	accountCfg := awscfg.Must(account.Config(ctx, mgmtCfg, roles.Administrator, time.Hour))
	networkCfg := awscfg.Must(mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour))
	substrateCfg := awscfg.Must(mgmtCfg.AssumeSubstrateRole(ctx, roles.Substrate, time.Hour))

	accounts.SetupIAM(ctx, mgmtCfg, networkCfg, substrateCfg, accountCfg, *domain, *environment, *quality)

	accounts.SetupTerraform(ctx, mgmtCfg, networkCfg, accountCfg, *domain, *environment, *quality)
	accounts.RunTerraform(*domain, *environment, *quality, *autoApprove, *noApply)

}
