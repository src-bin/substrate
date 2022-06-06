package versionutil

import (
	"context"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/tags"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

func PreventDowngrade(ctx context.Context, cfg *awscfg.Config) {
	t, err := cfg.Tags(ctx)
	if err != nil {
		ui.Fatal(err)
	}
	if taggedVersion := t[tags.SubstrateVersion]; taggedVersion > version.Version {
		ui.Fatalf(
			"your organization requires at least Substrate %v; exiting because this is Substrate %v",
			taggedVersion,
			version.Version,
		)
	} else if taggedVersion < version.Version {
		ui.Printf(
			"upgrading the minimum required Substrate version for your organization from %v to %v",
			taggedVersion,
			version.Version,
		)
	}
}
