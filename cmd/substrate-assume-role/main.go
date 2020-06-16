package main

import (
	"flag"
	"log"
	"os"
	"os/user"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

func main() {
	admin := flag.Bool("admin", false, `shorthand for -domain="admin" -environment="admin"`)
	domain := flag.String("domain", "", "domain of an AWS account in which to assume a role")
	environment := flag.String("environment", "", "environment of an AWS account in which to assume a role")
	quality := flag.String("quality", "", "quality of an AWS account in which to assume a role")
	special := flag.String("special", "", `name of a special AWS account in which to assume a role ("deploy", "master" or "network"`)
	master := flag.Bool("master", false, "assume a role in the organization's master AWS account")
	rolename := flag.String("role", "", "name of the IAM role to assume")
	flag.Parse()
	if *admin {
		*domain, *environment = "admin", "admin"
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" && !*master {
		ui.Fatal(`one of -domain="..." -environment="..." -quality"..." or -special="..." or -master is required`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *special != "" {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -special="..."`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *master {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -master`)
	}
	if *special != "" && *master {
		ui.Fatal(`can't mix -special="..." with -master`)
	}
	if *rolename == "" {
		ui.Fatal(`-role="..." is required`)
	}

	sess := awssessions.Must(awssessions.InMasterAccount(roles.OrganizationReader, awssessions.Config{}))
	svc := organizations.New(sess)
	var accountId string
	if *master {
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

	out, err := awssts.AssumeRole(sts.New(sess), roles.Arn(accountId, *rolename), u.Username)
	if err != nil {
		log.Fatal(err)
	}
	creds := out.Credentials

	if err := os.Setenv("AWS_ACCESS_KEY_ID", aws.StringValue(creds.AccessKeyId)); err != nil {
		log.Fatal(err)
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", aws.StringValue(creds.SecretAccessKey)); err != nil {
		log.Fatal(err)
	}
	if err := os.Setenv("AWS_SESSION_TOKEN", aws.StringValue(creds.SessionToken)); err != nil {
		log.Fatal(err)
	}

	awssts.Export(out, nil)

}
