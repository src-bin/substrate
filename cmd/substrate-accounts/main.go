package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/version"
)

func main() {
	cmdutil.MustChdir()
	flag.Parse()
	version.Flag()

	sess, err := awssessions.InManagementAccount(roles.OrganizationReader, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}
	svc := organizations.New(sess)

	if err := accounts.CheatSheet(svc); err != nil {
		log.Fatal(err)
	}
	f, err := os.Open(accounts.CheatSheetFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	io.Copy(os.Stdout, f)

}
