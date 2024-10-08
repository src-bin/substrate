package versionutil

import (
	"context"
	"os"
	"strings"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/contextutil"
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

// PreventDowngrade prevents explicit version number downgrades.
func PreventDowngrade(ctx context.Context, cfg *awscfg.Config) {

	if isUntaggedVersion(version.Version) {
		return
	}

	organizationVersion, ok := OrganizationVersion(ctx, cfg)
	if !ok {
		return
	}
	if isUntaggedVersion(organizationVersion) {
		return
	}

	switch Compare(organizationVersion, version.Version) {
	case Less:
		if subcommand := contextutil.ValueString(
			ctx,
			contextutil.Subcommand,
		); subcommand == "setup" || subcommand == "account adopt" || subcommand == "account create" || subcommand == "account update" {
			ui.Printf(
				"upgrading the minimum required Substrate version for your organization from %v to %v",
				organizationVersion,
				version.Version,
			)
		}
	case Greater:
		ui.Printf(
			"your organization requires at least Substrate %v; exiting because this is Substrate %v",
			organizationVersion,
			version.Version,
		)
		ui.Print("you should run `substrate upgrade`")
		os.Exit(1)
	}
}

// OrganizationVersion returns the SubstrateVersion tag from the calling
// account and true if that value is meaningful (i.e. non-empty). It returns
// the empty string and false if for whatever reason it can't read the tag.
func OrganizationVersion(ctx context.Context, cfg *awscfg.Config) (string, bool) {

	// If this is a test suite run, return early so that preventing downgrades
	// is never the reason a test suite fails.
	if executable, _ := os.Executable(); strings.HasSuffix(executable, ".test") {
		return "", false
	}

	t, err := cfg.Tags(ctx)
	if awsutil.ErrorCodeIs(err, awscfg.AWSOrganizationsNotInUseException) {
		return "", false // if we can't even fetch tags, we can't very well claim this is a downgrade
	}
	if awsutil.ErrorCodeIs(err, awscfg.AccessDenied) {
		return "", false // likewise if we can't assume OrganizationReader, it's also too early to claim it's a downgrade
	}
	if err != nil {
		ui.Fatal(err)
	}
	return t[tagging.SubstrateVersion], true
}

func WarnDowngrade(ctx context.Context, cfg *awscfg.Config) {

	if isUntaggedVersion(version.Version) {
		return
	}

	organizationVersion, ok := OrganizationVersion(ctx, cfg)
	if !ok {
		return
	}
	if isUntaggedVersion(organizationVersion) {
		return
	}

	if Compare(organizationVersion, version.Version) == Greater {
		ui.Printf("your organization has upgraded to Substrate %v; you should run `substrate upgrade`", organizationVersion)
	}
}

func isUntaggedVersion(v string) bool {
	if v == "1970.01" { // this is not even a build
		return true
	}
	if v = strings.TrimSuffix(v, "-dirty"); len(v) > 3 && v[len(v)-3] != '.' { // this is not a tagged build
		return true
	}
	return false
}
