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
	rolename := flag.String("role", "", "name of the IAM role to assume")
	flag.Parse()
	if *admin {
		*domain, *environment = "admin", "admin"
	}
	if (*domain == "" || *environment == "" || *quality == "") && *special == "" {
		ui.Fatal(`one of -domain="..." -environment="..." -quality"..." or -special="..." is required`)
	}
	if (*domain != "" || *environment != "" || *quality != "") && *special != "" {
		ui.Fatal(`can't mix -domain="..." -environment="..." -quality"..." with -special="..."`)
	}
	if *rolename == "" {
		ui.Fatal(`-role="..." is required`)
	}

	sess, err := awssessions.NewSession(awssessions.Config{})
	/*
		if err != nil {
			ui.Printf("unable to assume the role, which may mean this program is running outside of AWS; please provide an access key")
			accessKeyId, secretAccessKey := awsutil.ReadAccessKeyFromStdin()
			ui.Printf("using access key %s", accessKeyId)
			sess, err = awssessions.NewSession(awssessions.Config{
				AccessKeyId:     accessKeyId,
				SecretAccessKey: secretAccessKey,
			})
		}
	*/
	if err != nil {
		log.Fatal(err)
	}

	var account *organizations.Account
	if *special != "" {
		account, err = awsorgs.FindSpecialAccount(organizations.New(sess), *special)
	} else {
		account, err = awsorgs.FindAccount(organizations.New(sess), *domain, *environment, *quality)
	}
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
