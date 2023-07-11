package versionutil

import (
	"context"
	"os"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

// PreventSetupViolation prevents running commands like `substrate bootstrap-*`
// and `substrate create-admin-account` (or indeed any command that calls this
// function) once `substrate setup` has been run.
func PreventSetupViolation(ctx context.Context, cfg *awscfg.Config) {
	tags, err := cfg.Tags(ctx)

	// In PreventDowngrade, we silence errors from AWS, assuming they're
	// mostly lack of credentials and only experienced during initial
	// bootstrapping. That same logic holds here though the stakes are
	// potentially a little higher.
	if err != nil {
		return
	}
	// ui.Must(err)

	if tags[tagging.Name] == tagging.Substrate || tags[tagging.SubstrateType] != "" {
		ui.Printf(
			"your organization has upgraded to Substrate %s and `%s %s` is no longer available",
			contextutil.ValueString(ctx, contextutil.Command),
			contextutil.ValueString(ctx, contextutil.Subcommand),
			version.Version,
		)
		ui.Print("run `substrate setup` instead")
		os.Exit(1)
	}
}
