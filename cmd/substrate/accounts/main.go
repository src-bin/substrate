package accounts

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsorgs"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(cfg *awscfg.Main) {
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

	adminAccounts, serviceAccounts, auditAccount, deployAccount, managementAccount, networkAccount, err := accounts.Grouped(svc)
	if err != nil {
		log.Fatal(err)
	}
	switch format.String() {

	case cmdutil.SerializationFormatJSON:
		ui.PrettyPrintJSON(append(append([]*awsorgs.Account{
			managementAccount,
			auditAccount,
			networkAccount,
			deployAccount,
		}, adminAccounts...), serviceAccounts...))

	case cmdutil.SerializationFormatShell:
		fmt.Println("substrate-bootstrap-management-account")
		fmt.Println("substrate-bootstrap-network-account")
		fmt.Println("substrate-bootstrap-deploy-account")
		for _, account := range adminAccounts {
			fmt.Printf(
				"substrate-create-admin-account -quality=%q\n",
				account.Tags[tags.Quality],
			)
		}
		for _, account := range serviceAccounts {
			if _, ok := account.Tags[tags.Domain]; !ok {
				continue
			}
			fmt.Printf(
				"substrate-create-account -domain=%q -environment=%q -quality=%q\n",
				account.Tags[tags.Domain],
				account.Tags[tags.Environment],
				account.Tags[tags.Quality],
			)
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
