package main

import (
	"flag"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/src-bin/substrate/awsiam"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/choices"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/users"
	"github.com/src-bin/substrate/version"
)

func main() {
	cmdutil.Chdir()
	flag.Parse()
	version.Flag()

	sess := awssessions.Must(awssessions.InManagementAccount(roles.OrganizationAdministrator, awssessions.Config{}))

	ui.Spin("deleting all access keys for the OrganizationAdministrator user")
	if err := awsiam.DeleteAllAccessKeys(
		iam.New(sess),
		users.OrganizationAdministrator,
	); err != nil {
		log.Fatal(err)
	}
	ui.Stop("done")

	_, err := ui.Prompt("visit <https://console.aws.amazon.com/iam/home#/security_credentials> and delete all root access keys (which can't be deleted via the API) and press <enter>")
	if err != nil {
		log.Fatal(err)
	}

	intranetDNSDomainName, err := fileutil.ReadFile(choices.IntranetDNSDomainNameFilename)
	if err != nil {
		log.Fatal(err)
	}
	ui.Printf(
		"from now on, use <https://%s/credential-factory> to get temporary AWS access keys",
		strings.Trim(string(intranetDNSDomainName), "\r\n"),
	)

}
