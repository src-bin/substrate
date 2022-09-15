package versionutil

import (
	"context"
	"os"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/version"
)

type Comparison int

const (
	Less    Comparison = -1
	Equal   Comparison = 0
	Greater Comparison = 1
)

// Compare does a fairly standard three-state string comparison of two version
// numbers with two extensions. Firstly, it returns a type with a String()
// method. Secondly, and far more importantly, it only compares the first seven
// characters of each version number, which allows the second-resolution faux
// version numbers we set on development builds to compare equal to a release
// version from that same month.
func Compare(v1, v2 string) Comparison {
	if len(v1) >= 7 {
		v1 = v1[:7]
	}
	if len(v2) >= 7 {
		v2 = v2[:7]
	}
	if v1 > v2 {
		return Greater
	}
	if v1 < v2 {
		return Less
	}
	return Equal
}

func (cmp Comparison) String() string {
	switch cmp {
	case Less:
		return "Less"
	case Equal:
		return "Equal"
	case Greater:
		return "Greater"
	}
	panic(cmp)
}

func PreventDowngrade(ctx context.Context, cfg *awscfg.Config) {
	t, err := cfg.Tags(ctx)
	if awsutil.ErrorCodeIs(err, awscfg.AWSOrganizationsNotInUseException) {
		return // if we can't even fetch tags, we can't very well claim this is a downgrade
	}
	if awsutil.ErrorCodeIs(err, awscfg.AccessDenied) {
		return // likewise if we can't assume OrganizationReader, it's also too early to claim it's a downgrade
	}
	if err != nil {
		ui.Fatal(err)
	}
	switch taggedVersion := t[tagging.SubstrateVersion]; Compare(taggedVersion, version.Version) {
	case Less:
		ui.Printf(
			"upgrading the minimum required Substrate version for your organization from %v to %v",
			taggedVersion,
			version.Version,
		)
	case Greater:
		ui.Printf(
			"your organization requires at least Substrate %v; exiting because this is Substrate %v",
			taggedVersion,
			version.Version,
		)
		ui.Print("you should run `substrate upgrade`")
		os.Exit(1)
	}
}
