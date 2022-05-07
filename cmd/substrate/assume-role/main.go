package assumerole

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/federation"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	admin := flag.Bool("admin", false, `shorthand for -domain "admin" -environment "admin"`)
	domain := flag.String("domain", "", "domain of an AWS account in which to assume a role")
	environment := flag.String("environment", "", "environment of an AWS account in which to assume a role")
	quality := flag.String("quality", "", "quality of an AWS account in which to assume a role")
	special := flag.String("special", "", `name of a special AWS account in which to assume a role ("deploy", "management" or "network")`)
	management := flag.Bool("management", false, "assume a role in the organization's management AWS account")
	master := flag.Bool("master", false, "deprecated name for -management")
	number := flag.String("number", "", "account number of the AWS account in which to assume a role")
	roleName := flag.String("role", "", "name of the IAM role to assume")
	console := flag.Bool("console", false, "open the AWS Console to assume a role instead of generating an access key")
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatExportWithHistory) // default to undocumented special value for substrate-assume-role
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	oldpwd := cmdutil.MustChdir()
	flag.Usage = func() {
		ui.Print("Usage: substrate assume-role -management|-special <special> [-role <role>] [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		ui.Print("       substrate assume-role -admin -quality <quality> [-role <role>] [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		ui.Print("       substrate assume-role -domain <domain> -environment <environment -quality <quality> [-role <role>] [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		ui.Print("       substrate assume-role -number <number> [-role <role>] [-console] [-format <format>] [-quiet] [<command> [<argument> [...]]]")
		flag.PrintDefaults()
		ui.Print("  <command> [<argument> [...]]\n      command and optional arguments to invoke with the assumed role's credentials in its environment")
	}
	flag.Parse()
	*management = *management || *master
	version.Flag()
	if *admin {
		*domain, *environment = "admin", "admin"
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" && !*management && *number == "" {
		ui.Fatal(`one of -domain "..." -environment "..." -quality "..." or -special "..." or -management or -number "..." is required`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *special != "" {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality"..." with -special "..."`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *management {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality"..." with -management`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *number != "" {
		ui.Fatal(`can't mix -domain "..." -environment "..." -quality"..." with -number "..."`)
	}
	if *special != "" && *management {
		ui.Fatal(`can't mix -special "..." with -management`)
	}
	if *special != "" && *number != "" {
		ui.Fatal(`can't mix -special "..." with -number "..."`)
	}
	if *management && *number != "" {
		ui.Fatal(`can't mix -management with -number "..."`)
	}
	if *quiet {
		ui.Quiet()
	}

	callerIdentity, err := cfg.GetCallerIdentity(ctx)
	if err != nil {
		ui.Fatal(err)
	}
	currentRoleName, err := roles.Name(aws.ToString(callerIdentity.Arn))
	if err != nil {
		ui.Fatal(err)
	}

	if *number != "" {
		//accountId = *number // FIXME
		if *roleName == "" {
			ui.Fatal(`-role "..." is required with -number "..."`)
		}
		cfg, err = cfg.AssumeRole(ctx, *number, *roleName)
	} else if *management {
		if *roleName == "" {
			if currentRoleName == roles.Auditor {
				roleName = aws.String(roles.OrganizationReader)
			} else {
				roleName = aws.String(roles.OrganizationAdministrator)
			}
		}
		cfg, err = cfg.AssumeManagementRole(ctx, *roleName)
	} else if *special != "" {
		if *roleName == "" {
			if *special == "audit" || currentRoleName == roles.Auditor {
				roleName = aws.String(roles.Auditor)
			} else {
				roleName = aws.String(fmt.Sprintf("%s%s", strings.Title(*special), roles.Administrator))
			}
		}
		cfg, err = cfg.AssumeSpecialRole(ctx, *special, *roleName)
	} else {
		if *roleName == "" {
			if currentRoleName == roles.OrganizationAdministrator {
				roleName = aws.String(roles.Administrator)
			} else {
				roleName = aws.String(currentRoleName)
			}
		}
		cfg, err = cfg.AssumeServiceRole(ctx, *domain, *environment, *quality, *roleName)
	}
	if err != nil {
		ui.Fatal(err)
	}

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	credentials, err := cfg.Retrieve(ctx)
	if err != nil {
		ui.Fatal(err)
	}

	if *console {
		consoleSigninURL, err := federation.ConsoleSigninURL(credentials, "")
		if err != nil {
			log.Fatal(err)
		}
		ui.OpenURL(consoleSigninURL)
		return
	}

	// Execute a command with the credentials in its environment.  We use
	// os.Setenv instead of exec.Cmd.Env because we also want to preserve
	// other environment variables in case they're relevant to the command.
	if args := flag.Args(); len(args) > 0 {
		if err := os.Setenv("AWS_ACCESS_KEY_ID", credentials.AccessKeyID); err != nil {
			log.Fatal(err)
		}
		if err := os.Setenv("AWS_SECRET_ACCESS_KEY", credentials.SecretAccessKey); err != nil {
			log.Fatal(err)
		}
		if err := os.Setenv("AWS_SESSION_TOKEN", credentials.SessionToken); err != nil {
			log.Fatal(err)
		}

		// Switch back to the original working directory before looking for the
		// program to execute.
		if err := os.Chdir(oldpwd); err != nil {
			log.Fatal(err)
		}

		// Distinguish between a command error, which presumably is described
		// by the command itself before exiting with a non-zero status, and
		// command not found, which is our responsibility as the pseudo-shell.
		if _, err := exec.LookPath(flag.Args()[0]); err != nil {
			log.Fatal(err)
		}

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
	awssts.PrintCredentials(format, credentials)

}
