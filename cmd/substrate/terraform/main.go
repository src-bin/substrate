package terraform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmd/substrate/terraform/install"
	rootmodules "github.com/src-bin/substrate/cmd/substrate/terraform/root-modules"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/ui"
)

var (
	domain, domainFlag, domainCompletionFunc                = cmdutil.DomainFlag("domain of an AWS account in which to run Terraform")
	environment, environmentFlag, environmentCompletionFunc = cmdutil.EnvironmentFlag("environment of an AWS account in which to run Terraform")
	quality, qualityFlag, qualityCompletionFunc             = cmdutil.QualityFlag("quality of an AWS account in which to run Terraform")
	special                                                 = new(string)
	substrate                                               = new(bool)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: `terraform --domain <domain> --environment <environment> [--quality <quality>] init|plan|apply|... [...]",
  substrate terraform install|root-modules`,
		Short: "TODO terraform.Command().Short",
		Long:  `TODO terraform.Command().Long`,
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {

			// Once we're past the account selection arguments, defer to
			// Terraform's own autocomplete, which is not very good. But if it
			// gets better, we'll be ready!
			if (*domain != "" && *environment != "") || *special != "" || *substrate {
				b := &bytes.Buffer{}
				cmd := exec.Command("terraform")
				cmd.Env = append(
					os.Environ(),
					fmt.Sprintf("COMP_LINE=terraform %s ", strings.Join(args, " ")),
				)
				cmd.Stdout = b
				ui.Must(cmd.Run())
				return fileutil.ToLines(b.Bytes()), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
			}

			return []string{
				"--domain", "--environment", "--quality",
				"--special", "--substrate",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().AddFlag(domainFlag)
	cmd.RegisterFlagCompletionFunc(domainFlag.Name, domainCompletionFunc)
	cmd.Flags().AddFlag(environmentFlag)
	cmd.RegisterFlagCompletionFunc(environmentFlag.Name, environmentCompletionFunc)
	cmd.Flags().AddFlag(qualityFlag)
	cmd.RegisterFlagCompletionFunc(qualityFlag.Name, qualityCompletionFunc)
	cmd.Flags().StringVar(special, "special", "", `name of a special AWS account in which to assume a role ("deploy" or "network")`)
	cmd.RegisterFlagCompletionFunc("special", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"deploy", "network"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(substrate, "substrate", false, "assume a role in the AWS organization's Substrate account")

	cmd.AddCommand(install.Command())
	cmd.AddCommand(rootmodules.Command())

	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, _ []string, _ io.Writer) {
	panic("not implemented")
}
