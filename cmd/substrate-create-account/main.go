package main

import (
	"flag"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

func main() {
	domain := flag.String("domain", "", "Domain for this new AWS account")
	environment := flag.String("environment", "", "Environment for this new AWS account")
	quality := flag.String("quality", "", "Quality for this new AWS account")
	flag.Parse()
	if *domain == "" || *environment == "" || *quality == "" {
		ui.Print(`-domain="..." -environment="..." -quality"..." are required`)
		os.Exit(1)
	}

	sess, err := awssessions.InMasterAccount(roles.OrganizationAdministrator, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}

	ui.Spin("finding or creating the account")
	account, err := awsorgs.EnsureAccount(organizations.New(svc), *domain, *environment, *quality)
	if err != nil {
		log.Fatal(err)
	}
	ui.Stopf("account %s", account.Id)
	log.Printf("%+v", account)

	// TODO add this account to the principals of the OrganizationReader role

}
