package adopt

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
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/versionutil"
)

var (
	number                                                  = new(string)
	domain, domainFlag, domainCompletionFunc                = cmdutil.DomainFlag("domain to assign to this AWS account")
	environment, environmentFlag, environmentCompletionFunc = cmdutil.EnvironmentFlag("environment to assign to this AWS account")
	quality, qualityFlag, qualityCompletionFunc             = cmdutil.QualityFlag("quality to assign to this AWS account")
	ignoreServiceQuotas                                     = new(bool)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adopt --number <number> --domain <domain> --environment <environment> [--quality <quality>]",
		Short: "adopt an AWS account in your organization into Substrate management",
		Long:  ``,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--number",
				"--domain", "--environment", "--quality",
				"--ignore-service-quotas",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().StringVar(number, "number", "", "account number of the AWS account in your organization to adopt into Substrate management")
	cmd.RegisterFlagCompletionFunc("number", cmdutil.NoCompletionFunc)
	cmd.Flags().AddFlag(domainFlag)
	cmd.RegisterFlagCompletionFunc(domainFlag.Name, domainCompletionFunc)
	cmd.Flags().AddFlag(environmentFlag)
	cmd.RegisterFlagCompletionFunc(environmentFlag.Name, environmentCompletionFunc)
	cmd.Flags().AddFlag(qualityFlag)
	cmd.RegisterFlagCompletionFunc(qualityFlag.Name, qualityCompletionFunc)
	cmd.Flags().BoolVar(ignoreServiceQuotas, "ignore-service-quotas", false, "ignore the appearance of any service quota being exhausted and continue anyway")
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {
	if *number == "" {
		ui.Fatal(`--number "..." is required`)
	}
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

	mgmtCfg := awscfg.Must(cfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	versionutil.PreventDowngrade(ctx, mgmtCfg)

	ui.Spin("finding the account")
	account, err := awsorgs.DescribeAccount(ctx, mgmtCfg, *number)
	if awsutil.ErrorCodeIs(err, awsorgs.AccountNotFoundException) {
		ui.Stop("not found")
		ui.Printf("is account number %s a member of your organization?", *number)
		os.Exit(1)
	}
	ui.Must(err)
	ui.Stop(account)
	if account.Tags[tagging.Manager] == tagging.Substrate {
		ui.Printf("%s is already being managed by Substrate", account)
		os.Exit(1)
	}

	mgmtCfg.Telemetry().FinalAccountId = aws.ToString(account.Id)
	mgmtCfg.Telemetry().FinalRoleName = roles.Administrator

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	accountCfg := awscfg.Must(account.Config(ctx, mgmtCfg, roles.Administrator, time.Hour))
	networkCfg := awscfg.Must(mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour))
	substrateCfg := awscfg.Must(mgmtCfg.AssumeSubstrateRole(ctx, roles.Substrate, time.Hour))

	accounts.SetupIAM(ctx, mgmtCfg, networkCfg, substrateCfg, accountCfg, *domain, *environment, *quality)

	accounts.SetupTerraform(ctx, mgmtCfg, networkCfg, accountCfg, *domain, *environment, *quality)

	ui.Print("next, commit the following files to version control:")
	ui.Print("")
	ui.Print("substrate.*")
	ui.Printf("modules/%s/", *domain)
	ui.Print("modules/common/")
	ui.Print("modules/substrate/")
	ui.Printf("root-modules/%s/%s/%s/", *domain, *environment, *quality)
	ui.Print("")
	ui.Printf(
		"then, write Terraform code in modules/%s/ to define your infrastructure and run `substrate account update`, `substrate terraform`, or plain `terraform` to apply it",
		*domain,
	)

}
