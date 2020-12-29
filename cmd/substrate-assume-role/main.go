package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/user"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func main() {
	admin := flag.Bool("admin", false, `shorthand for -domain="admin" -environment="admin"`)
	domain := flag.String("domain", "", "domain of an AWS account in which to assume a role")
	environment := flag.String("environment", "", "environment of an AWS account in which to assume a role")
	quality := flag.String("quality", "", "quality of an AWS account in which to assume a role")
	special := flag.String("special", "", `name of a special AWS account in which to assume a role ("deploy", "master" or "network"`)
	master := flag.Bool("master", false, "assume a role in the organization's master AWS account")
	number := flag.String("number", "", "account number of the AWS account in which to assume a role")
	rolename := flag.String("role", "", "name of the IAM role to assume")
	quiet := flag.Bool("quiet", false, "do not write anything to standard output before forking the child command")
	flag.Parse()
	version.Flag()
	if *admin {
		*domain, *environment = "admin", "admin"
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" && !*master && *number == "" {
		ui.Fatal(`one of -domain="..." -environment="..." -quality"..." or -special="..." or -master or -number="..." is required`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *special != "" {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -special="..."`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *master {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -master`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *number != "" {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -number="..."`)
	}
	if *special != "" && *master {
		ui.Fatal(`can't mix -special="..." with -master`)
	}
	if *special != "" && *number != "" {
		ui.Fatal(`can't mix -special="..." with -number="..."`)
	}
	if *master && *number != "" {
		ui.Fatal(`can't mix -master with -number="..."`)
	}
	if *rolename == "" {
		ui.Fatal(`-role="..." is required`)
	}
	if *quiet {
		ui.Quiet()
	}

	sess, err := awssessions.InMasterAccount(roles.OrganizationReader, awssessions.Config{})
	if err != nil {
		ui.Fatal(awssessions.NewOrganizationReaderError(err, *rolename))
	}
	svc := organizations.New(sess)
	var accountId string
	if *number != "" {
		accountId = *number
	} else if *master {
		org, err := awsorgs.DescribeOrganization(svc)
		if err != nil {
			log.Fatal(err)
		}
		accountId = aws.StringValue(org.MasterAccountId)
	} else if *special != "" {
		accountId = aws.StringValue(awsorgs.Must(awsorgs.FindSpecialAccount(svc, *special)).Id)
	} else {
		accountId = aws.StringValue(awsorgs.Must(awsorgs.FindAccount(svc, *domain, *environment, *quality)).Id)
	}

	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	sess = awssessions.Must(awssessions.NewSession(awssessions.Config{}))

	out, err := awssts.AssumeRole(
		sts.New(sess),
		roles.Arn(accountId, *rolename),
		u.Username,
		3600, // AWS-enforced maximum when crossing accounts per <https://aws.amazon.com/premiumsupport/knowledge-center/iam-role-chaining-limit/>
	)
	if err != nil {
		log.Fatal(err)
	}
	creds := out.Credentials

	// Print the credentials for the user to copy into their environment.
	if !*quiet {
		awssts.Export(out, nil)
	}

	// Execute a command with the credentials in its environment.  We use
	// os.Setenv instead of exec.Cmd.Env because we also want to preserve
	// other environment variables in case they're relevant to the command.
	if args := flag.Args(); len(args) > 0 {
		if err := os.Setenv("AWS_ACCESS_KEY_ID", aws.StringValue(creds.AccessKeyId)); err != nil {
			log.Fatal(err)
		}
		if err := os.Setenv("AWS_SECRET_ACCESS_KEY", aws.StringValue(creds.SecretAccessKey)); err != nil {
			log.Fatal(err)
		}
		if err := os.Setenv("AWS_SESSION_TOKEN", aws.StringValue(creds.SessionToken)); err != nil {
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
	}

}
