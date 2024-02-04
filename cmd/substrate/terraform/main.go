package terraform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmd/substrate/terraform/install"
	rootmodules "github.com/src-bin/substrate/cmd/substrate/terraform/root-modules"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
)

var (
	domain, domainFlag, domainCompletionFunc                = cmdutil.DomainFlag("domain of an AWS account in which to run Terraform")
	environment, environmentFlag, environmentCompletionFunc = cmdutil.EnvironmentFlag("environment of an AWS account in which to run Terraform")
	quality, qualityFlag, qualityCompletionFunc             = cmdutil.QualityFlag("quality of an AWS account in which to run Terraform")
	special                                                 = new(string)
	substrate                                               = new(bool)
	global                                                  = new(bool)
	region                                                  = new(string)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: `terraform --domain <domain> --environment <environment> [--quality <quality>] [--global|--region <region>] init|plan|apply|... [...]
  substrate terraform --special <special> [--global|--region <region>] init|plan|apply|... [...]
  substrate terraform --substrate [--global|--region <region>] init|plan|apply|... [...]`,
		Short: "run Terraform in a specific AWS account",
		Long:  ``,
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {

			// Once we're past the account selection arguments, defer to
			// Terraform's own autocomplete, which is not very good. But if it
			// gets better, we'll be ready!
			if (*domain != "" && *environment != "" || *special != "" || *substrate) && (*global || *region != "") {
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
	cmd.Flags().StringVar(special, "special", "", `name of a special AWS account in which to run Terraform ("deploy" or "network")`)
	cmd.RegisterFlagCompletionFunc("special", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"deploy", "network"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(substrate, "substrate", false, "run Terraform in the AWS organization's Substrate account")
	cmd.Flags().BoolVarP(global, "global", "g", false, "run Terraform in a global root module")
	cmd.Flags().StringVarP(region, "region", "r", "", "name of the region in which to run Terraform")
	cmd.RegisterFlagCompletionFunc("region", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return regions.Selected(), cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().SetInterspersed(false)

	cmd.AddCommand(install.Command())
	cmd.AddCommand(rootmodules.Command())

	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, args []string, _ io.Writer) {
	if *environment != "" && *quality == "" {
		*quality = cmdutil.QualityForEnvironment(*environment)
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" && !*substrate {
		ui.Fatal(`one of --domain "..." --environment "..." --quality "..." or --special "..." or --substrate is required`)
	}
	if *domain != "" && *special != "" {
		ui.Fatal(`can't mix --domain "..." with --special "..."`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *substrate {
		ui.Fatal(`can't mix --domain "..." --environment "..." --quality "..." with --substrate`)
	}
	if *special != "" && *substrate {
		ui.Fatal(`can't mix --special "..." with --substrate`)
	}
	if !*global && *region == "" {
		ui.Fatal("one of --global or --region \"...\" is required; use `substrate account update` to run all of an account's root modules")
	}

	dirname := terraform.RootModulesDirname
	if *domain != "" && *environment != "" && *quality != "" {
		dirname = filepath.Join(dirname, *domain, *environment, *quality)
	} else if *special != "" {
		switch *special {
		case accounts.Deploy:
			dirname = filepath.Join(dirname, accounts.Deploy)
		case accounts.Network:
			if *environment == "" || *quality == "" {
				ui.Fatal(`--environment "..." is required with --special "network"`)
			}
			if *quality == "" {
				ui.Fatal(`--quality "..." is required with --special "network"`)
			}
			dirname = filepath.Join(dirname, accounts.Network, *environment, *quality)
		default:
			ui.Fatalf("--special %q is invalid", special)
		}
	} else if *substrate {
		if *quality == "" {
			substrateAccount := ui.Must2(cfg.FindSubstrateAccount(ctx))
			*quality = ui.Must2(substrateAccount.Quality())
		}
		dirname = filepath.Join(dirname, naming.Admin, *quality)
	}
	if *global {
		dirname = filepath.Join(dirname, regions.Global)
	} else {
		dirname = filepath.Join(dirname, *region)
	}
	//ui.PrintWithCaller(dirname)

	cmd := exec.Command("terraform", args...)
	cmd.Dir = dirname
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	//ui.PrintfWithCaller("%+v", cmd)
	if err := cmd.Run(); err != nil {
		ui.Fatal(err)
	}
}
