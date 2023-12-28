package assumerole

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/versionutil"
)

var (
	domain, domainFlag, domainCompletionFunc                = cmdutil.DomainFlag("domain of an AWS account in which to assume a role")
	environment, environmentFlag, environmentCompletionFunc = cmdutil.EnvironmentFlag("environment of an AWS account in which to assume a role")
	quality, qualityFlag, qualityCompletionFunc             = cmdutil.QualityFlag("quality of an AWS account in which to assume a role")
	management                                              = new(bool)
	special                                                 = new(string)
	substrate                                               = new(bool)
	number                                                  = new(string)
	roleName, roleARN                                       = new(string), new(string)
	console                                                 = new(bool)
	format, formatFlag, formatCompletionFunc                = cmdutil.FormatFlag(
		cmdutil.FormatExportWithHistory,
		[]cmdutil.Format{cmdutil.FormatEnv, cmdutil.FormatExport, cmdutil.FormatExportWithHistory, cmdutil.FormatJSON},
	)
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: `assume-role --domain <domain> --environment <environment> [--quality <quality>] [--role <role-name>] [--console] [--format <format>] [--quiet] [<command> [<argument> [...]]]
  substrate assume-role --management|--special <special>|--substrate [--role <role-name>] [--console] [--format <format>] [--quiet] [<command> [<argument> [...]]]
  substrate assume-role --number <number> --role <role-name> [--console] [--format <format>] [--quiet] [<command> [<argument> [...]]]
  substrate assume-role --arn <role-arn> [--console] [--format <format>] [--quiet] [<command> [<argument> [...]]]`,
		Short: "assume a role in another AWS account",
		Long:  ``,
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Main(cmdutil.Main(cmd, args))
		},
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{
				"--domain", "--environment", "--quality",
				"--management", "--special", "--substrate",
				"--number",
				"--role", "--arn",
				"--console",
				"--format",
				"--quiet",
			}, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
		},
	}
	cmd.Flags().AddFlag(domainFlag)
	cmd.RegisterFlagCompletionFunc(domainFlag.Name, domainCompletionFunc)
	cmd.Flags().AddFlag(environmentFlag)
	cmd.RegisterFlagCompletionFunc(environmentFlag.Name, environmentCompletionFunc)
	cmd.Flags().AddFlag(qualityFlag)
	cmd.RegisterFlagCompletionFunc(qualityFlag.Name, qualityCompletionFunc)
	cmd.Flags().BoolVar(management, "management", false, "assume a role in the AWS organization's management account")
	cmd.Flags().StringVar(special, "special", "", `name of a special AWS account in which to assume a role ("audit", "deploy", or "network")`)
	cmd.RegisterFlagCompletionFunc("special", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"audit", "deploy", "network"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().BoolVar(substrate, "substrate", false, "assume a role in the AWS organization's Substrate account")
	cmd.Flags().StringVar(number, "number", "", "account number of the AWS account in which to assume a role")
	cmd.RegisterFlagCompletionFunc("number", cmdutil.NoCompletionFunc)
	cmd.Flags().StringVar(roleName, "role", "", "name of the IAM role to assume")
	cmd.RegisterFlagCompletionFunc("role", cmdutil.NoCompletionFunc)
	cmd.Flags().StringVar(roleARN, "arn", "", "ARN of the IAM role to assume")
	cmd.RegisterFlagCompletionFunc("arn", cmdutil.NoCompletionFunc)
	cmd.Flags().BoolVar(console, "console", false, "open the AWS Console to assume a role instead of generating an access key")
	cmd.Flags().AddFlag(formatFlag)
	cmd.RegisterFlagCompletionFunc(formatFlag.Name, formatCompletionFunc)
	cmd.Flags().AddFlag(cmdutil.QuietFlag())
	return cmd
}

func Main(ctx context.Context, cfg *awscfg.Config, _ *cobra.Command, args []string, _ io.Writer) {
	if *environment != "" && *quality == "" {
		*quality = cmdutil.QualityForEnvironment(*environment)
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" && !*substrate && !*management && *number == "" && *roleARN == "" {
		ui.Fatal(`one of --domain "..." --environment "..." --quality "..." or --management or --special "..." or --substrate or --number "..." or --arn "..." is required`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *management {
		ui.Fatal(`can't mix --domain "..." --environment "..." --quality "..." with --management`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *special != "" {
		ui.Fatal(`can't mix --domain "..." --environment "..." --quality "..." with --special "..."`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *substrate {
		ui.Fatal(`can't mix --domain "..." --environment "..." --quality "..." with --substrate`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *number != "" {
		ui.Fatal(`can't mix --domain "..." --environment "..." --quality "..." with --number "..."`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *roleARN != "" {
		ui.Fatal(`can't mix --domain "..." --environment "..." --quality "..." with --arn "..."`)
	}
	if *management && *special != "" {
		ui.Fatal(`can't mix --management with --special "..."`)
	}
	if *management && *substrate {
		ui.Fatal(`can't mix --management with --substrate`)
	}
	if *management && *number != "" {
		ui.Fatal(`can't mix --management with --number "..."`)
	}
	if *management && *roleARN != "" {
		ui.Fatal(`can't mix --management with --arn "..."`)
	}
	if *special != "" && *substrate {
		ui.Fatal(`can't mix --special "..." with --substrate`)
	}
	if *special != "" && *number != "" {
		ui.Fatal(`can't mix --special "..." with --number "..."`)
	}
	if *special != "" && *roleARN != "" {
		ui.Fatal(`can't mix --special "..." with --arn "..."`)
	}
	if *substrate && *number != "" {
		ui.Fatal(`can't mix --substrate with --number "..."`)
	}
	if *substrate && *roleARN != "" {
		ui.Fatal(`can't mix --substrate with --arn "..."`)
	}

	callerIdentity := cfg.MustGetCallerIdentity(ctx)
	currentRoleName, err := roles.Name(aws.ToString(callerIdentity.Arn))
	ui.Must(err)
	duration := time.Hour

	versionutil.WarnDowngrade(ctx, cfg)

	// Do the dance to get 12-hour credentials in the current role so that we
	// can get 12-hour credentials for the final role, too.
	// TODO this might not actually be possible, depending on the current role
	// TODO maybe make it optional or only with -console, SLOOOOW
	// TODO THE REASON THIS DOESN'T WORK is that it bundles a janky same-account-only AssumeRole; we need to pass it accountId, too, and have it replace the call below
	/*
		ui.Spin("minting temporary credentials that last 12 hours")
		creds, err := awsiam.AllDayCredentials(ctx, cfg, aws.ToString(callerIdentity.Account), currentRoleName)
		if err != nil {
			ui.Fatal(err)
		}
		if _, err := cfg.SetCredentials(ctx, creds); err != nil {
			ui.Fatal(err)
		}
		duration = 11 * time.Hour // XXX 12; 11 is a test
		ui.Stop("ok")
		ci, err := cfg.GetCallerIdentity(ctx)
	*/

	if *roleARN != "" {
		cfg, err = cfg.AssumeRoleARN(ctx, *roleARN, duration)
	} else if *number != "" {
		if *roleName == "" {
			ui.Fatal(`--role "..." is required with --number "..."`)
		}
		cfg, err = cfg.AssumeRole(ctx, *number, *roleName, duration)
	} else if *substrate {
		if *roleName == "" {
			if currentRoleName == roles.OrganizationAdministrator {
				roleName = aws.String(roles.Administrator)
			} else {
				roleName = aws.String(currentRoleName)
			}
		}
		cfg, err = cfg.AssumeSubstrateRole(ctx, *roleName, duration)
	} else if *management {
		if *roleName == "" {
			if currentRoleName == roles.Auditor {
				roleName = aws.String(roles.OrganizationReader)
			} else {
				roleName = aws.String(roles.OrganizationAdministrator)
			}
		}
		cfg, err = cfg.AssumeManagementRole(ctx, *roleName, duration)
	} else if *special != "" {
		if *roleName == "" {
			if currentRoleName == roles.Auditor {
				roleName = aws.String(roles.Auditor)
			} else {
				roleName = aws.String(fmt.Sprintf("%s%s", strings.Title(*special), roles.Administrator))
			}
		}
		cfg, err = cfg.AssumeSpecialRole(ctx, *special, *roleName, duration)
	} else {
		if *roleName == "" {
			if currentRoleName == roles.OrganizationAdministrator {
				roleName = aws.String(roles.Administrator)
			} else {
				roleName = aws.String(currentRoleName)
			}
		}
		cfg, err = cfg.AssumeServiceRole(ctx, *domain, *environment, *quality, *roleName, duration)
	}
	if err != nil {
		ui.Print(err)
		if os.Getenv("OLD_AWS_ACCESS_KEY_ID") != "" {
			ui.Print("this might be because you already assumed a role; run `unassume-role` and try again")
		}
		os.Exit(1)
	}

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier
	defer cfg.Telemetry().Wait(ctx)

	creds, err := cfg.Retrieve(ctx)
	if err != nil {
		ui.Fatal(err)
	}

	if *console {
		consoleSigninURL, err := federation.ConsoleSigninURL(
			creds,
			"", // destination (empty means the AWS Console homepage)
			nil,
		)
		if err != nil {
			ui.Fatal(err)
		}
		ui.OpenURL(consoleSigninURL)
		return
	}

	// Execute a command with the credentials in its environment.  We use
	// os.Setenv instead of exec.Cmd.Env because we also want to preserve
	// other environment variables in case they're relevant to the command.
	if len(args) > 0 {
		ui.Must(awscfg.Setenv(creds))

		// Switch back to the original working directory before looking for the
		// program to execute.
		ui.Must(cmdutil.UndoChdir())

		// Distinguish between a command error, which presumably is described
		// by the command itself before exiting with a non-zero status, and
		// command not found, which is our responsibility as the pseudo-shell.
		_, err := exec.LookPath(args[0])
		ui.Must(err)

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			os.Exit(1)
		}

		return
	}

	// Print the credentials for the user to copy into their environment.
	cmdutil.PrintCredentials(*format, creds)

}
