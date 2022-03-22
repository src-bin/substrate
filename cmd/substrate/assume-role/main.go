package assumerole

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(cfg *awscfg.Main) {
	admin := flag.Bool("admin", false, `shorthand for -domain="admin" -environment="admin"`)
	domain := flag.String("domain", "", "domain of an AWS account in which to assume a role")
	environment := flag.String("environment", "", "environment of an AWS account in which to assume a role")
	quality := flag.String("quality", "", "quality of an AWS account in which to assume a role")
	special := flag.String("special", "", `name of a special AWS account in which to assume a role ("deploy", "management" or "network"`)
	management := flag.Bool("management", false, "assume a role in the organization's management AWS account")
	master := flag.Bool("master", false, "deprecated name for -management")
	number := flag.String("number", "", "account number of the AWS account in which to assume a role")
	roleName := flag.String("role", "", "name of the IAM role to assume")
	console := flag.Bool("console", false, "open the AWS Console to assume a role instead of generating an access key")
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatExportWithHistory) // default to undocumented special value for substrate-assume-role
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	oldpwd := cmdutil.MustChdir()
	flag.Parse()
	*management = *management || *master
	version.Flag()
	if *admin {
		*domain, *environment = "admin", "admin"
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" && !*management && *number == "" {
		ui.Fatal(`one of -domain="..." -environment="..." -quality"..." or -special="..." or -management or -number="..." is required`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *special != "" {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -special="..."`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *management {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -management`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *number != "" {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -number="..."`)
	}
	if *special != "" && *management {
		ui.Fatal(`can't mix -special="..." with -management`)
	}
	if *special != "" && *number != "" {
		ui.Fatal(`can't mix -special="..." with -number="..."`)
	}
	if *management && *number != "" {
		ui.Fatal(`can't mix -management with -number="..."`)
	}
	if *quiet {
		ui.Quiet()
	}

	sess := awssessions.Must(awssessions.NewSession(awssessions.Config{}))
	svc := sts.New(sess)
	callerIdentity, err := awssts.GetCallerIdentity(svc)
	if err != nil {
		ui.Fatal(err)
	}
	currentRoleName, err := roles.Name(aws.StringValue(callerIdentity.Arn))
	if err != nil {
		ui.Fatal(err)
	}

	var accountId string
	{
		sess, err := awssessions.InManagementAccount(roles.OrganizationReader, awssessions.Config{})
		if err != nil {

			// Mask the AWS-native error because we're 99% sure OrganizationReaderError
			// is a better explanation of what went wrong.
			if _, ok := err.(awserr.Error); ok {
				ui.Fatal(awssessions.NewOrganizationReaderError(err, *roleName))
			}

			ui.Fatal(err)
		}
		svc := organizations.New(sess)
		if *number != "" {
			accountId = *number
			if *roleName == "" {
				ui.Fatal(`-role="..." is required with -number="..."`)
			}
		} else if *management {
			org, err := awsorgs.DescribeOrganization(svc)
			if err != nil {
				log.Fatal(err)
			}
			accountId = aws.StringValue(org.MasterAccountId)
			if *roleName == "" {
				if currentRoleName == roles.Auditor {
					roleName = aws.String(roles.OrganizationReader)
				} else {
					roleName = aws.String(roles.OrganizationAdministrator)
				}
			}
		} else if *special != "" {
			accountId = aws.StringValue(awsorgs.Must(awsorgs.FindSpecialAccount(svc, *special)).Id)
			if *roleName == "" {
				if *special == "audit" || currentRoleName == roles.Auditor {
					roleName = aws.String(roles.Auditor)
				} else {
					roleName = aws.String(fmt.Sprintf("%s%s", strings.Title(*special), roles.Administrator))
				}
			}
		} else {
			accountId = aws.StringValue(awsorgs.Must(awsorgs.FindAccount(svc, *domain, *environment, *quality)).Id)
			if *roleName == "" {
				if currentRoleName == roles.OrganizationAdministrator {
					roleName = aws.String(roles.Administrator)
				} else {
					roleName = aws.String(currentRoleName)
				}
			}
		}
	}

	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	assumedRole, err := awssts.AssumeRole(
		svc,
		roles.Arn(accountId, *roleName),
		u.Username,
		3600, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/> // TODO 43200?
	)
	if err != nil {
		log.Fatal(err)
	}
	credentials := assumedRole.Credentials

	if *console {
		consoleSigninURL, err := awssts.ConsoleSigninURL(svc, credentials, "")
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
		if err := os.Setenv("AWS_ACCESS_KEY_ID", aws.StringValue(credentials.AccessKeyId)); err != nil {
			log.Fatal(err)
		}
		if err := os.Setenv("AWS_SECRET_ACCESS_KEY", aws.StringValue(credentials.SecretAccessKey)); err != nil {
			log.Fatal(err)
		}
		if err := os.Setenv("AWS_SESSION_TOKEN", aws.StringValue(credentials.SessionToken)); err != nil {
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
