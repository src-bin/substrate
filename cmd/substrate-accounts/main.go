package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func main() {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatText) // default to undocumented special value
	cmdutil.MustChdir()
	flag.Parse()
	version.Flag()

	sess, err := awssessions.InManagementAccount(roles.OrganizationReader, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}
	svc := organizations.New(sess)

	// Update substrate.accounts.txt unconditionally as this is the expected
	// side-effect of running this program.
	if err := accounts.CheatSheet(svc); err != nil {
		log.Fatal(err)
	}

	switch format.String() {

	case cmdutil.SerializationFormatJSON:
		adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(svc)
		if err != nil {
			log.Fatal(err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "\t")
		if err := enc.Encode(append(append([]*awsorgs.Account{
			managementAccount,
			auditAccount,
			networkAccount,
			deployAccount,
		}, adminAccounts...), serviceAccounts...)); err != nil {
			ui.Fatal(err)
		}

	case cmdutil.SerializationFormatText:
		f, err := os.Open(accounts.CheatSheetFilename)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		io.Copy(os.Stdout, f)

	default:
		ui.Fatalf("-format=%q not supported", format)
	}

}
