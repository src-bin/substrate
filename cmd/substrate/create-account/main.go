package createaccount

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/humans"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	create := flag.Bool("create", false, "create a new AWS account, if necessary, without confirmation")
	domain := cmdutil.DomainFlag("domain for this new AWS account")
	environment := cmdutil.EnvironmentFlag("environment for this new AWS account")
	ignoreServiceQuotas := flag.Bool("ignore-service-quotas", false, "ignore the appearance of any service quota being exhausted and continue anyway")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	number := flag.String("number", "", "tag and begin managing this account instead of creating a new AWS account")
	quality := cmdutil.QualityFlag("quality for this new AWS account")
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate create-account [-create|-number <number>] -domain <domain> -environment <environment> [-quality <quality>] [-auto-approve|-no-apply] [-ignore-service-quotas]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
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
	var account *awsorgs.Account
	if *number == "" {
		account, err = mgmtCfg.FindServiceAccount(ctx, *domain, *environment, *quality)
		ui.Must(err)
		if account == nil {
			ui.Stop("not found")
			if !*create {
				if ok, err := ui.Confirm("create a new AWS account? (yes/no)"); err != nil {
					ui.Fatal(err)
				} else if !ok {
					ui.Fatal("not creating a new AWS account")
				}
			}
			ui.Spin("creating the account")
			var deadline time.Time
			if *ignoreServiceQuotas {
				deadline = time.Now()
			}
			account, err = awsorgs.EnsureAccount(
				ctx,
				mgmtCfg,
				*domain,
				*environment,
				*quality,
				deadline,
			)
		}
	} else {
		account, err = awsorgs.DescribeAccount(ctx, mgmtCfg, *number)
	}
	ui.Must(err)
	ui.Must(awsorgs.Tag(
		ctx,
		mgmtCfg,
		aws.ToString(account.Id),
		tagging.Map{
			tagging.Domain:      *domain,
			tagging.Environment: *environment,
			tagging.Manager:     tagging.Substrate,
			//tagging.Name: awsorgs.NameFor(*domain, *environment, *quality), // don't override this in case it was an invited account with an important name
			tagging.Quality:          *quality,
			tagging.SubstrateVersion: version.Version,
		},
	))
	ui.Must(accounts.CheatSheet(ctx, awscfg.Must(mgmtCfg.OrganizationReader(ctx))))
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	mgmtCfg.Telemetry().FinalAccountId = aws.ToString(account.Id)
	mgmtCfg.Telemetry().FinalRoleName = roles.Administrator

	ui.Spin("configuring IAM")
	accountCfg, err := account.Config(ctx, mgmtCfg, roles.Administrator, time.Hour)
	if err != nil {
		accountCfg, err = account.Config(ctx, mgmtCfg, roles.OrganizationAccountAccessRole, time.Hour)
	}
	ui.Must(err)
	ui.Must2(humans.EnsureAdministratorRole(ctx, mgmtCfg, accountCfg))
	ui.Must2(humans.EnsureAuditorRole(ctx, mgmtCfg, accountCfg))
	networkCfg := awscfg.Must(mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour))

	// TODO if the legacy network account exists, ensure there's a network for this service account there
	// TODO if not, create (with confirmation) a network account for this environment and quality, peer it, and pass it along

	ui.Must2(terraform.EnsureStateManager(ctx, awscfg.Must(mgmtCfg.AssumeSubstrateRole(ctx, roles.Substrate, time.Hour))))

	// Create CloudWatch's cross-account sharing role in this account.
	//
	// This probably shouldn't be a core part of Substrate but it has been
	// for longer than Substrate had custom role management and would be
	// a bit troublesome to remove now.
	const cloudwatchRoleName = "CloudWatch-CrossAccountSharingRole"
	orgAssumeRolePolicy, err := awsorgs.OrgAssumeRolePolicy(ctx, mgmtCfg)
	ui.Must(err)
	cloudwatchRole, err := awsiam.EnsureRole(ctx, accountCfg, cloudwatchRoleName, orgAssumeRolePolicy)
	ui.Must(err)
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/job-function/ViewOnlyAccess"))
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/AWSXrayReadOnlyAccess"))
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/CloudWatchReadOnlyAccess"))
	ui.Must(awsiam.AttachRolePolicy(ctx, accountCfg, cloudwatchRole.Name, "arn:aws:iam::aws:policy/CloudWatchAutomaticDashboardsAccess"))
	ui.Stop("ok")

	// Leave the user a place to put their own Terraform code that can be
	// shared between all of a domain's accounts.
	ui.Must(terraform.Scaffold(*domain, true))

	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect every AWS region in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each region")
	}
	{
		dirname := filepath.Join(terraform.RootModulesDirname, *domain, *environment, *quality, regions.Global)
		region := regions.Default()

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Label: terraform.Q(*domain),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.UsEast1ProviderAlias: terraform.UsEast1ProviderAlias,
			},
			Source: terraform.Q("../../../../../modules/", *domain, "/global"),
		})
		ui.Must(file.WriteIfNotExists(filepath.Join(dirname, "main.tf")))

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
		))
		providersFile.Add(terraform.UsEast1Provider(
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))

		ui.Must(terraform.Init(dirname))

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		ui.Must(err)
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, *domain, *environment, *quality, region)

		networks.ShareVPC(
			ctx,
			accountCfg.Regional(region),
			networkCfg.Regional(region),
			*domain, *environment, *quality,
			region,
		)
		ui.Must(fileutil.Remove(filepath.Join(dirname, "network.tf")))

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Label: terraform.Q(*domain),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.NetworkProviderAlias: terraform.NetworkProviderAlias,
			},
			Source: terraform.Q("../../../../../modules/", *domain, "/regional"),
		})
		ui.Must(file.WriteIfNotExists(filepath.Join(dirname, "main.tf")))

		// Even though we say in main.tf that it won't be overwritten, we need
		// to selectively overwrite it to remove the depends_on attribute that
		// used to order the customer's code to come after the VPC subnets
		// were definitely shared because they're no longer in Terraform.
		b, err := os.ReadFile(filepath.Join(dirname, "main.tf"))
		ui.Must(err)
		b = regexp.MustCompile(
			` *depends_on = \[
( *aws_ec2_tag\.[0-9a-z-]*,
)* *\]
`,
		).ReplaceAllLiteral(b, []byte{})
		ui.Must(os.WriteFile(filepath.Join(dirname, "main.tf"), b, 0666))

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.ARN(aws.ToString(account.Id), roles.Administrator),
		))
		providersFile.Add(terraform.NetworkProviderFor(
			region,
			roles.ARN(networkCfg.MustAccountId(ctx), roles.NetworkAdministrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))

		ui.Must(terraform.Init(dirname))

		if *noApply {
			if err := terraform.Plan(dirname); err != nil {
				ui.Print(err) // allow these plans to fail and keep going to accommodate folks who keep certain regions' networks destroyed
			}
		} else {
			ui.Must(terraform.Apply(dirname, *autoApprove))
		}
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Print("next, commit the following files to version control:")
	ui.Print("")
	ui.Print("substrate.*")
	ui.Printf("modules/%s/", *domain)
	ui.Print("modules/common/")
	ui.Print("modules/substrate/")
	ui.Printf("root-modules/%s/%s/%s/", *domain, *environment, *quality)
	ui.Print("")
	ui.Printf(
		"then, write Terraform code in modules/%s/ to define the rest of your infrastructure or run `substrate create-account` again for other domains, environments, and/or qualities",
		*domain,
	)

}

func Synopsis() {
	panic("not implemented")
}
