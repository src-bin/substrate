package accounts

import (
	"context"
	"os"
	"path/filepath"
	"regexp"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
)

func RunTerraform(
	domain, environment, quality string,
	autoApprove, noApply bool,
) {
	if !autoApprove && !noApply {
		ui.Print("this tool can affect every AWS region in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each region")
	}

	{
		dirname := filepath.Join(terraform.RootModulesDirname, domain, environment, quality, regions.Global)

		ui.Must(terraform.Init(dirname))
		ui.Must(terraform.ProvidersLock(dirname))

		if noApply {
			ui.Must(terraform.Plan(dirname))
		} else {
			ui.Must(terraform.Apply(dirname, autoApprove))
		}
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, domain, environment, quality, region)

		ui.Must(terraform.Init(dirname))
		ui.Must(terraform.ProvidersLock(dirname))

		// Remove network sharing and tagging from Terraform because Substrate
		// handles that directly now.
		networks.StateRm(dirname, domain, environment, quality, region)

		if noApply {
			if err := terraform.Plan(dirname); err != nil {
				ui.Print(err) // allow these plans to fail and keep going to accommodate folks who keep certain regions' networks destroyed
			}
		} else {
			ui.Must(terraform.Apply(dirname, autoApprove))
		}
	}

	if noApply {
		ui.Print("-no-apply given so not invoking `terraform apply`")
	}
}

func SetupTerraform(
	ctx context.Context,
	mgmtCfg, networkCfg, accountCfg *awscfg.Config,
	domain, environment, quality string,
) {

	// Leave the user a place to put their own Terraform code that can be
	// shared between all of a domain's accounts.
	ui.Must(terraform.Scaffold(domain, true))

	accountId := accountCfg.MustAccountId(ctx)
	{
		dirname := filepath.Join(terraform.RootModulesDirname, domain, environment, quality, regions.Global)
		region := regions.Default()

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Label: terraform.Q(domain),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.UsEast1ProviderAlias: terraform.UsEast1ProviderAlias,
			},
			Source: terraform.Q("../../../../../modules/", domain, "/global"),
		})
		ui.Must(file.WriteIfNotExists(filepath.Join(dirname, "main.tf")))

		providersFile := terraform.NewFile()
		providersFile.Add(terraform.ProviderFor(
			region,
			roles.ARN(accountId, roles.Administrator),
		))
		providersFile.Add(terraform.UsEast1Provider(
			roles.ARN(accountId, roles.Administrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))
	}
	for _, region := range regions.Selected() {
		dirname := filepath.Join(terraform.RootModulesDirname, domain, environment, quality, region)

		networks.ShareVPC(
			ctx,
			accountCfg.Regional(region),
			networkCfg.Regional(region),
			domain, environment, quality,
			region,
		)
		ui.Must(fileutil.Remove(filepath.Join(dirname, "network.tf")))

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Label: terraform.Q(domain),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.DefaultProviderAlias: terraform.DefaultProviderAlias,
				terraform.NetworkProviderAlias: terraform.NetworkProviderAlias,
			},
			Source: terraform.Q("../../../../../modules/", domain, "/regional"),
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
			roles.ARN(accountId, roles.Administrator),
		))
		providersFile.Add(terraform.NetworkProviderFor(
			region,
			roles.ARN(networkCfg.MustAccountId(ctx), roles.NetworkAdministrator),
		))
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

		ui.Must(terraform.Fmt(dirname))
	}

}
