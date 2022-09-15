package whoami

import (
	"context"
	"flag"
	"fmt"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
)

func Main(ctx context.Context, cfg *awscfg.Config) {
	format := cmdutil.SerializationFormatFlag(cmdutil.SerializationFormatText) // default to undocumented special value
	quiet := flag.Bool("quiet", false, "suppress status and diagnostic output")
	flag.Usage = func() {
		ui.Print("Usage: substrate whoami [-format <format>] [-quiet]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *quiet {
		ui.Quiet()
	}

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	identity, err := cfg.Identity(ctx)
	if err != nil {
		ui.Fatal(err)
	}

	switch format.String() {
	case cmdutil.SerializationFormatEnv:
		fmt.Printf(
			"DOMAIN=%q\nENVIRONMENT=%q\nQUALITY=%q\nROLE=%q\n",
			identity.Tags.Domain,
			identity.Tags.Environment,
			identity.Tags.Quality,
			identity.ARN,
		)
	case cmdutil.SerializationFormatExport, cmdutil.SerializationFormatExportWithHistory:
		fmt.Printf(
			"export DOMAIN=%q ENVIRONMENT=%q QUALITY=%q ROLE=%q\n",
			identity.Tags.Domain,
			identity.Tags.Environment,
			identity.Tags.Quality,
			identity.ARN,
		)
	case cmdutil.SerializationFormatJSON:
		ui.PrettyPrintJSON(map[string]string{
			tagging.Domain:      identity.Tags.Domain,
			tagging.Environment: identity.Tags.Environment,
			tagging.Quality:     identity.Tags.Quality,
			"Role":              identity.ARN,
		})
	case cmdutil.SerializationFormatText:
		ui.Printf(
			"you're %s in\nDomain:      %s\nEnvironment: %s\nQuality:     %s",
			identity.ARN,
			identity.Tags.Domain,
			identity.Tags.Environment,
			identity.Tags.Quality,
		)
	default:
		ui.Fatalf("-format %q not supported", format)
	}

}
