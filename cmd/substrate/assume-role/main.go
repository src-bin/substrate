package assumerole

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	admin := flag.Bool("admin", false, `(deprecated) shorthand for -domain "admin" -environment "admin"`)
	domain := cmdutil.DomainFlag("domain of an AWS account in which to assume a role")
	environment := cmdutil.EnvironmentFlag("environment of an AWS account in which to assume a role")
	quality := cmdutil.QualityFlag("quality of an AWS account in which to assume a role")
	special := flag.String("special", "", `name of a special AWS account in which to assume a role ("deploy" or "network")`)
	substrate := flag.Bool("substrate", false, "assume a role in the AWS organization's Substrate account")
	management := flag.Bool("management", false, "assume a role in the AWS organization's management account")
	master := flag.Bool("master", false, "deprecated name for -management")
	number := flag.String("number", "", "account number of the AWS account in which to assume a role")
	roleName := flag.String("role", "", "name of the IAM role to assume")
	roleARN := flag.String("arn", "", "ARN of the IAM role to assume")
	console := flag.Bool("console", false, "open the AWS Console to assume a role instead of generating an access key")
	format := cmdutil.SerializationFormatFlag(
		cmdutil.SerializationFormatExportWithHistory, // default to undocumented special value for substrate-assume-role
		cmdutil.SerializationFormatUsage,
	)
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	flag.Usage = func() {
		ui.Print("Usage: substrate assume-role -management|-special <special>|-substrate [-role <role-name>] [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		ui.Print("       substrate assume-role -domain <domain> -environment <environment> [-quality <quality>] [-role <role-name>] [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		ui.Print("       substrate assume-role -number <number> -role <role-name> [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		ui.Print("       substrate assume-role -arn <role-arn> [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		flag.PrintDefaults()
		ui.Print("  <command> [<argument> [...]]\n      command and optional arguments to invoke with the assumed role's credentials in its environment")
	}
	flag.Parse()
	*management = *management || *master
	version.Flag()
	if *admin {
		*domain, *environment = "admin", "admin"
	}
	if *environment != "" && *quality == "" {
		*quality = cmdutil.QualityForEnvironment(*environment)
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" && !*substrate && !*management && *number == "" && *roleARN == "" {
		ui.Fatal(`one of -domain "..." -environment "..." -quality "..." or -management or -special "..." or -substrate or -number "..." or -arn "..." is required`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *management {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality "..." with -management`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *special != "" {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality "..." with -special "..."`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *substrate {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality "..." with -substrate`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *number != "" {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality "..." with -number "..."`)
	}
	if (*domain != "" || *environment != "" /* || *quality != "" */) && *roleARN != "" {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality "..." with -arn "..."`)
	}
	if *management && *special != "" {
		ui.Fatal(`can't mix -management with -special "..."`)
	}
	if *management && *substrate {
		ui.Fatal(`can't mix -management with -substrate`)
	}
	if *management && *number != "" {
		ui.Fatal(`can't mix -management with -number "..."`)
	}
	if *management && *roleARN != "" {
		ui.Fatal(`can't mix -management with -arn "..."`)
	}
	if *special != "" && *substrate {
		ui.Fatal(`can't mix -special "..." with -substrate`)
	}
	if *special != "" && *number != "" {
		ui.Fatal(`can't mix -special "..." with -number "..."`)
	}
	if *special != "" && *roleARN != "" {
		ui.Fatal(`can't mix -special "..." with -arn "..."`)
	}
	if *substrate && *number != "" {
		ui.Fatal(`can't mix -substrate with -number "..."`)
	}
	if *substrate && *roleARN != "" {
		ui.Fatal(`can't mix -substrate with -arn "..."`)
	}
	if *quiet {
		ui.Quiet()
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
			ui.Fatal(`-role "..." is required with -number "..."`)
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
			if *special == "audit" || currentRoleName == roles.Auditor {
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
	if args := flag.Args(); len(args) > 0 {
		ui.Must(os.Setenv("AWS_ACCESS_KEY_ID", creds.AccessKeyID))
		ui.Must(os.Setenv("AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey))
		ui.Must(os.Setenv("AWS_SESSION_TOKEN", creds.SessionToken))

		// Switch back to the original working directory before looking for the
		// program to execute.
		ui.Must(cmdutil.UndoChdir())

		// Distinguish between a command error, which presumably is described
		// by the command itself before exiting with a non-zero status, and
		// command not found, which is our responsibility as the pseudo-shell.
		_, err := exec.LookPath(flag.Args()[0])
		ui.Must(err)

		cmd := exec.Command(flag.Args()[0], flag.Args()[1:]...)
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
	cmdutil.PrintCredentials(format, creds)

}
