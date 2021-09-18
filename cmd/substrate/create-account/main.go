package createaccount

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/admin"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
	"github.com/src-bin/substrate/version"
)

func Main() {
	create := flag.Bool("create", false, "create a new AWS account, if necessary, without confirmation")
	domain := flag.String("domain", "", "domain for this new AWS account")
	environment := flag.String("environment", "", "environment for this new AWS account")
	quality := flag.String("quality", "", "quality for this new AWS account")
	autoApprove := flag.Bool("auto-approve", false, "apply Terraform changes without waiting for confirmation")
	noApply := flag.Bool("no-apply", false, "do not apply Terraform changes")
	cmdutil.MustChdir()
	flag.Parse()
	version.Flag()
	if *domain == "" || *environment == "" || *quality == "" {
		ui.Fatal(`-domain="..." -environment="..." -quality"..." are required`)
	}
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	if !veqpDoc.Valid(*environment, *quality) {
		ui.Fatalf(`-environment="%s" -quality="%s" is not a valid environment and quality pair in your organization`, *environment, *quality)
	}

	sess, err := awssessions.InManagementAccount(roles.OrganizationAdministrator, awssessions.Config{
		FallbackToRootCredentials: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	ui.Spin("finding the account")
	var account *awsorgs.Account
	{
		svc := organizations.New(sess)
		account, err = awsorgs.FindAccount(svc, *domain, *environment, *quality)
		if _, ok := err.(awsorgs.AccountNotFound); ok {
			ui.Stop("not found")
			if !*create {
				if ok, err := ui.Confirm("create a new AWS account? (yes/no)"); err != nil {
					log.Fatal(err)
				} else if !ok {
					ui.Fatal("not creating a new AWS account")
				}
			}
			ui.Spin("creating the account")
			account, err = awsorgs.EnsureAccount(svc, *domain, *environment, *quality)
		}
		if err != nil {
			log.Fatal(err)
		}
		if err := accounts.CheatSheet(svc); err != nil {
			log.Fatal(err)
		}
	}
	ui.Stopf("account %s", account.Id)
	//log.Printf("%+v", account)

	admin.EnsureAdminRolesAndPolicies(sess)

	// Leave the user a place to put their own Terraform code that can be
	// shared between all of a domain's accounts.
	if err := terraform.Scaffold(*domain); err != nil {
		log.Fatal(err)
	}

	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect every AWS region in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each region")
	}
	{
		dirname := filepath.Join(terraform.RootModulesDirname, *domain, *environment, *quality, regions.Global)
		region := "us-east-1"

		file := terraform.NewFile()
		file.Push(terraform.Module{
			Label:  terraform.Q(*domain),
			Source: terraform.Q("../../../../../modules/", *domain, "/global"),
		})
		if err := file.WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			log.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Push(terraform.ProviderFor(
			region,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
		))
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Root(dirname, region); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, *autoApprove)
		}
		if err != nil {
			log.Fatal(err)
		}
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, *domain, *environment, *quality, region)

		file := terraform.NewFile()
		file.Push(terraform.Module{
			Label: terraform.Q(*domain),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.NetworkProviderAlias: terraform.NetworkProviderAlias,
			},
			Source: terraform.Q("../../../../../modules/", *domain, "/regional"),
		})
		if err := file.WriteIfNotExists(filepath.Join(dirname, "main.tf")); err != nil {
			log.Fatal(err)
		}

		networkFile := terraform.NewFile()
		networks.ShareVPC(networkFile, account, *domain, *environment, *quality, region)
		if err := networkFile.Write(filepath.Join(dirname, "network.tf")); err != nil {
			log.Fatal(err)
		}

		providersFile := terraform.NewFile()
		providersFile.Push(terraform.ProviderFor(
			region,
			roles.Arn(aws.StringValue(account.Id), roles.Administrator),
		))
		networkAccount, err := awsorgs.FindSpecialAccount(organizations.New(awssessions.Must(awssessions.InManagementAccount(
			roles.OrganizationReader,
			awssessions.Config{},
		))), accounts.Network)
		if err != nil {
			log.Fatal(err)
		}
		providersFile.Push(terraform.NetworkProviderFor(
			region,
			roles.Arn(aws.StringValue(networkAccount.Id), roles.NetworkAdministrator), // TODO a role that only allows sharing VPCs would be a nice safety measure here
		))
		if err := providersFile.Write(filepath.Join(dirname, "providers.tf")); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Root(dirname, region); err != nil {
			log.Fatal(err)
		}

		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}

		if *noApply {
			if err := terraform.Plan(dirname); err != nil {
				ui.Print(err) // allow these plans to fail and keep going to accommodate folks who keep certain regions' networks destroyed
			}
		} else {
			if err := terraform.Apply(dirname, *autoApprove); err != nil {
				log.Fatal(err)
			}
		}
	}
	if *noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}

	ui.Printf(
		"next, commit modules/substrate/ and root-modules/%s/%s/%s/ to version control, then write Terraform code there to define the rest of your infrastructure or run substrate-create-account again for other domains, environments, and/or qualities",
		*domain,
		*environment,
		*quality,
	)

}
