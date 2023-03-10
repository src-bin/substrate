package createterraformmodule

import (
	"context"
	"flag"
	"io"
	"log"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	flag.Usage = func() {
		ui.Print("Usage: substrate create-terraform-module <name> [...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	version.Flag()
	if flag.NArg() == 0 {
		ui.Fatal("need at least one module name to create")
	}

	go cfg.Telemetry().Post(ctx) // post earlier, finish earlier

	for _, name := range flag.Args() {
		if err := terraform.Scaffold(name, true); err != nil {
			log.Fatal(err)
		}
	}
}
