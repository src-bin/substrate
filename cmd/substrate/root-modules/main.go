package rootmodules

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

func Main() {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatText) // default to undocumented special value // TODO only support text and JSON
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	flag.Parse()
	if *quiet {
		ui.Quiet()
	}

	sess, err := awssessions.InManagementAccount(roles.OrganizationReader, awssessions.Config{})
	if err != nil {
		log.Fatal(err)
	}
	svc := organizations.New(sess)
	adminAccounts, serviceAccounts, _, _, _, _, err := accounts.Grouped(svc)
	if err != nil {
		log.Fatal(err)
	}

	var rootModules []string

	// The management and audit accounts don't run Terraform so they aren't
	// mentioned in this program's output.

	// Deploy account.
	rootModules = append(rootModules, filepath.Join(
		terraform.RootModulesDirname,
		accounts.Deploy,
		regions.Global,
	))
	for _, region := range regions.Selected() {
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			accounts.Deploy,
			region,
		))
	}

	// Network account.
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region := range regions.Selected() {
			rootModules = append(rootModules, filepath.Join(
				terraform.RootModulesDirname,
				accounts.Network,
				eq.Environment,
				eq.Quality,
				region,
			))
		}
	}
	peeringConnections, err := networks.EnumeratePeeringConnections()
	if err != nil {
		log.Fatal(err)
	}
	for pc := range peeringConnections {
		eq0, eq1, region0, region1 := pc.Ends()
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			accounts.Network,
			"peering",
			eq0.Environment,
			eq1.Environment,
			eq0.Quality,
			eq1.Quality,
			region0,
			region1,
		))
	}

	// Admin accounts.
	for _, account := range adminAccounts {
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			accounts.Admin,
			account.Tags[tags.Quality],
			regions.Global,
		))
		for _, region := range regions.Selected() {
			rootModules = append(rootModules, filepath.Join(
				terraform.RootModulesDirname,
				accounts.Admin,
				account.Tags[tags.Quality],
				region,
			))
		}
	}

	for _, account := range serviceAccounts {
		if _, ok := account.Tags[tags.Domain]; !ok {
			continue
		}
		rootModules = append(rootModules, filepath.Join(
			terraform.RootModulesDirname,
			account.Tags[tags.Domain],
			account.Tags[tags.Environment],
			account.Tags[tags.Quality],
			regions.Global,
		))
		for _, region := range regions.Selected() {
			rootModules = append(rootModules, filepath.Join(
				terraform.RootModulesDirname,
				account.Tags[tags.Domain],
				account.Tags[tags.Environment],
				account.Tags[tags.Quality],
				region,
			))
		}
	}

	switch format.String() {
	case cmdutil.SerializationFormatJSON:
		ui.PrettyPrintJSON(rootModules)
	case cmdutil.SerializationFormatText:
		for _, rootModule := range rootModules {
			ui.Print(rootModule)
		}
	default:
		ui.Fatalf("-format=%q not supported", format)
	}

}
