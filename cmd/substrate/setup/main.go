package setup

import (
	"context"
	"flag"
	"io"
	"log"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/telemetry"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
	"github.com/src-bin/substrate/versionutil"
)

func Main(ctx context.Context, cfg *awscfg.Config, w io.Writer) {
	ui.InteractivityFlags()
	flag.Usage = func() {
		ui.Print("Usage: substrate setup")
		flag.PrintDefaults()
	}
	flag.Parse()

	if version.IsTrial() {
		ui.Print("since this is a trial version of Substrate, it will post non-sensitive and non-personally identifying telemetry (documented in more detail at <https://docs.src-bin.com/substrate/ref/telemetry>) to Source & Binary to better understand how Substrate is being used; paying customers may opt out of this telemetry")
	} else {
		_, err := ui.ConfirmFile(
			telemetry.Filename,
			"can Substrate post non-sensitive and non-personally identifying telemetry (documented in more detail at <https://docs.src-bin.com/substrate/ref/telemetry>) to Source & Binary to better understand how Substrate is being used? (yes/no)",
		)
		ui.Must(err)
	}

	if _, err := cfg.GetCallerIdentity(ctx); err != nil {
		if _, err := cfg.SetRootCredentials(ctx); err != nil {
			ui.Fatal(err)
		}
	}
	cfg = awscfg.Must(cfg.AssumeManagementRole(
		ctx,
		roles.Substrate,
		time.Hour, // XXX longer would be better since bootstrapping's expected to take some serious time
	))

	log.Print(jsonutil.MustString(cfg.MustGetCallerIdentity(ctx)))

	versionutil.PreventDowngrade(ctx, cfg)
	versionutil.PreventSetupViolation(ctx, cfg)

	prefix := naming.Prefix()

	region := regions.Default()
	cfg = cfg.Regional(region)

	_ = prefix

	// TODO maybe ask about telemetry

	// TODO create the organization, if necessary

	// TODO find or create and tag the audit account, if requested

	// TODO configure CloudTrail, if requested

	// TODO tag the management account

	// TODO find or create the network account (because we're probably not going to get all of this done before August 1)

	// TODO tag the deploy and network accounts, if they exist

	// TODO Service Control Policy (or perhaps punt to a whole new `substrate create-scp|scps` family of commands; also tagging policies)

	// TODO find the admin account or create the Substrate account

	// TODO tag the admin or Substrate account to make it unambiguously the Substrate account

	// TODO create the Substrate user in the Substrate account

	// TODO create the Substrate role in the Substrate account

	// TODO create the Substrate role in the management account

	// TODO create the TerraformStateManager role in the Substrate account

	// TODO ??? create legacy {Organization,Deploy,Network}Administrator and OrganizationReader roles ???

	// TODO create Administrator and Auditor roles in the Substrate account and every service account

	// TODO run the legacy deploy account's Terraform code, if the account exists

	// TODO run the legacy network account's Terraform code, if the account exists

	// TODO configure the Intranet

	// TODO configure IAM Identity Center (later)

	// TODO instructions on using the Credential Factory, Intranet, etc.

}
