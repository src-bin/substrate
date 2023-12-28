package terraform

import (
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/cmd/substrate/terraform/install"
	rootmodules "github.com/src-bin/substrate/cmd/substrate/terraform/root-modules"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: `terraform --domain <domain> --environment <environment> [--quality <quality>] init|plan|apply|... [...]",
  substrate terraform install|root-modules`,
		Short: "TODO terraform.Command().Short",
		Long:  `TODO terraform.Command().Long`,
	}
	// TODO

	cmd.AddCommand(install.Command())
	cmd.AddCommand(rootmodules.Command())

	return cmd
}
