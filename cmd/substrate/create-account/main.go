package createaccount

import (
	"context"
	"flag"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/cmdutil"
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

func Main(ctx context.Context, cfg *awscfg.Config) {
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	create := flag.Bool("create", false, "create a new AWS account, if necessary, without confirmation")
	domain := flag.String("domain", "", "domain for this new AWS account")
	environment := flag.String("environment", "", "environment for this new AWS account")
	ignoreServiceQuotas := flag.Bool("ignore-service-quotas", false, "ignore the appearance of any service quota being exhausted and continue anyway")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	quality := flag.String("quality", "", "quality for this new AWS account")
	cmdutil.MustChdir()
	flag.Usage = func() {
		ui.Print("Usage: substrate create-account [-create] -domain <domain> -environment <environment> -quality <quality> [-auto-approve|-no-apply] [-ignore-service-quotas]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if *domain == "" || *environment == "" || *quality == "" {
		ui.Fatal(`-domain "..." -environment "..." -quality"..." are required`)
	}
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		ui.Fatal(err)
	}
	if !veqpDoc.Valid(*environment, *quality) {
		ui.Fatalf(`-environment %q -quality %q is not a valid environment and quality pair in your organization`, *environment, *quality)
	}

	cfg = awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.OrganizationAdministrator,
		time.Hour,
	))
	versionutil.PreventDowngrade(ctx, cfg)

	ui.Spin("finding the account")
	var account *awsorgs.Account
	createdAccount := false
	{
		account, err = cfg.FindServiceAccount(ctx, *domain, *environment, *quality)
		if err != nil {
			ui.Fatal(err)
		}
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
				cfg,
				*domain,
				*environment,
				*quality,
				deadline,
			)
			createdAccount = true
		} else {
			err = awsorgs.Tag(
				ctx,
				cfg,
				aws.ToString(account.Id),
				tagging.Map{tagging.SubstrateVersion: version.Version},
			)
		}
		if err != nil {
			ui.Fatal(err)
		}
		if err := accounts.CheatSheet(ctx, awscfg.Must(cfg.OrganizationReader(ctx))); err != nil {
			ui.Fatal(err)
		}
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	cfg.Telemetry().FinalAccountId = aws.ToString(account.Id)
	cfg.Telemetry().FinalRoleName = roles.Administrator

	admin.EnsureAdminRolesAndPolicies(ctx, cfg, createdAccount)

	// Leave the user a place to put their own Terraform code that can be
	// shared between all of a domain's accounts.
	if err := terraform.Scaffold(*domain); err != nil {
		ui.Fatal(err)
	}

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
		if err := file.WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			ui.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.Arn(aws.ToString(account.Id), roles.Administrator),
		))
		providersFile.Add(terraform.UsEast1Provider(
			roles.Arn(aws.ToString(account.Id), roles.Administrator),
		))
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			ui.Fatal(err)
		}

		if err := terraform.Root(ctx, cfg, dirname, region); err != nil {
			ui.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			ui.Fatal(err)
		}

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		if err != nil {
			ui.Fatal(err)
		}
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, *domain, *environment, *quality, region)

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Label: terraform.Q(*domain),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.NetworkProviderAlias: terraform.NetworkProviderAlias,
			},
			Source: terraform.Q("../../../../../modules/", *domain, "/regional"),
		})
		if err := file.WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			ui.Fatal(err)
		}

		networkFile := terraform.NewFile()
		networks.ShareVPC(networkFile, account, *domain, *environment, *quality, region)
		if err := networkFile.Write(filepath.Join(dirname, "network.tf")); err != nil {
			ui.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.Arn(aws.ToString(account.Id), roles.Administrator),
		))
		networkAccount, err := cfg.FindSpecialAccount(ctx, accounts.Network)
		if err != nil {
			ui.Fatal(err)
		}
		providersFile.Add(terraform.NetworkProviderFor(
			region,
			roles.Arn(aws.ToString(networkAccount.Id), roles.NetworkAdministrator), // TODO a role that only allows sharing VPCs would be a nice safety measure here
		))
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			ui.Fatal(err)
		}

		if err := terraform.Root(ctx, cfg, dirname, region); err != nil {
			ui.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			ui.Fatal(err)
		}

		if *noApply {
			if err := terraform.Plan(dirname); err != nil {
				ui.Print(err) // allow these plans to fail and keep going to accommodate folks who keep certain regions' networks destroyed
			}
		} else {
			if err := terraform.Apply(dirname, *autoApprove); err != nil {
				ui.Fatal(err)
			}
		}
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Print("next, commit the following files to version control:")
	ui.Print("")
	ui.Print("substrate.*")
	ui.Printf("modules/%s/", *domain)
	ui.Print("modules/substrate/")
	ui.Printf("root-modules/%s/%s/%s/", *domain, *environment, *quality)
	ui.Print("")
	ui.Printf(
		"then, write Terraform code in modules/%s/ to define the rest of your infrastructure or run `substrate create-account` again for other domains, environments, and/or qualities",
		*domain,
	)

}
