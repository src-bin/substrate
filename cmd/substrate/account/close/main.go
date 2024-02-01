package close

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/versionutil"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	domain, domainFlag, domainCompletionFunc                = cmdutil.DomainFlag("domain of the AWS account to close")
	environment, environmentFlag, environmentCompletionFunc = cmdutil.EnvironmentFlag("environment of the AWS account to close")
	quality, qualityFlag, qualityCompletionFunc             = cmdutil.QualityFlag("quality of the AWS account to close")
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close --domain <domain> --environment <environment> [--quality <quality>]",
		Short: "close an AWS account",
		Long:  ``,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--domain", "--environment", "--quality",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().AddFlag(domainFlag)
	cmd.RegisterFlagCompletionFunc(domainFlag.Name, domainCompletionFunc)
	cmd.Flags().AddFlag(environmentFlag)
	cmd.RegisterFlagCompletionFunc(environmentFlag.Name, environmentCompletionFunc)
	cmd.Flags().AddFlag(qualityFlag)
	cmd.RegisterFlagCompletionFunc(qualityFlag.Name, qualityCompletionFunc)
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
	if strings.ContainsAny(*domain, ", ") {
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

	cmdutil.PrintRoot()

	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(ctx, roles.OrganizationAdministrator, time.Hour))
	versionutil.PreventDowngrade(ctx, mgmtCfg)

	// Confirm really aggressively that they really want to close this account.
	if !terminal.IsTerminal(0) || !terminal.IsTerminal(1) || !terminal.IsTerminal(2) {
		ui.Fatal("standard input, output, and error must be attached to a TTY to use `substrate account close`")
	}
	accountCfg := awscfg.Must(cfg.AssumeServiceRole(ctx, *domain, *environment, *quality, roles.Auditor, time.Hour))
	identity := ui.Must2(accountCfg.Identity(ctx))
	ui.Printf(
		"are you sure you want to close this service account?\nDomain:      %s\nEnvironment: %s\nQuality:     %s",
		identity.Tags.Domain,
		identity.Tags.Environment,
		identity.Tags.Quality,
	)
	if !ui.Must2(ui.Confirmf("close account number %s? (yes/no)", accountCfg.MustAccountId(ctx))) {
		return
	}
	ui.Print("")
	ui.Print("there's a limited period in which you can tediously undo this action")
	ui.Printf(
		"are you really, really sure you want to close this service account?\nDomain:      %s\nEnvironment: %s\nQuality:     %s",
		identity.Tags.Domain,
		identity.Tags.Environment,
		identity.Tags.Quality,
	)
	if !ui.Must2(ui.Confirmf("close account number %s? (yes/no)", accountCfg.MustAccountId(ctx))) {
		return
	}

	ui.Spinf("closing account number %s", accountCfg.MustAccountId(ctx))
	time.Sleep(5e9) // give them a chance to ^C
	ui.Must(awsorgs.CloseAccount(
		ctx,
		mgmtCfg,
		accountCfg.MustAccountId(ctx),
	))
	ui.Must(mgmtCfg.ClearCachedAccounts())
	ui.Stop("ok")

	go mgmtCfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer mgmtCfg.Telemetry().Wait(ctx)

	ui.Print("")
	ui.Print("next, consider whether to remove the following files from version control:")
	ui.Print("")
	ui.Printf("modules/%s/", *domain)
	ui.Printf("root-modules/%s/%s/%s/", *domain, *environment, *quality)

}
