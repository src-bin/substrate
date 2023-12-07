package adoptaccount

import (
	"context"
	"flag"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	number := flag.String("number", "", "tag and begin managing this account instead of creating a new AWS account")
	domain := cmdutil.DomainFlag("domain for this new AWS account")
	environment := cmdutil.EnvironmentFlag("environment for this new AWS account")
	quality := cmdutil.QualityFlag("quality for this new AWS account")
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate adopt-account -number <number> -domain <domain> -environment <environment> [-quality <quality>]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *number == "" {
		ui.Fatal(`-number "..." is required`)
	}
	if *environment != "" && *quality == "" {
		*quality = cmdutil.QualityForEnvironment(*environment)
	}
	if *domain == "" || *environment == "" || *quality == "" {
		ui.Fatal(`-domain "..." -environment "..." -quality"..." are required`)
	}
	if d := *domain; d == "admin" || d == "common" || d == "deploy" || d == "intranet" || d == "lambda-function" || d == "network" || d == "peering-connection" || d == "substrate" {
		ui.Fatalf(`-domain %q is reserved; please choose a different name`, d)
	}
	if strings.ContainsAny(*domain, ", ") {
		ui.Fatalf("-domain %q cannot contain commas or spaces", *domain)
	}
	if strings.ContainsAny(*environment, ", ") {
		ui.Fatalf("-environment %q cannot contain commas or spaces", *environment)
	}
	if strings.ContainsAny(*quality, ", ") {
		ui.Fatalf("-quality %q cannot contain commas or spaces", *quality)
	}
	veqpDoc, err := veqp.ReadDocument()
	ui.Must(err)
	if !veqpDoc.Valid(*environment, *quality) {
		ui.Fatalf(`-environment %q -quality %q is not a valid environment and quality pair in your organization`, *environment, *quality)
	}

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
		"then, write Terraform code in modules/%s/ to define your infrastructure and run `substrate update-account` to apply it",
		*domain,
	)

}

func Synopsis() {
	panic("not implemented")
}
