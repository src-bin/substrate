package testawscfg

import (
	"context"
	"time"

	"github.com/src-bin/substrate/awscfg"
)

const (
	Test1AdminAccountId = "716893237583"
	Test2AdminAccountId = "944106955638"
	Test3AdminAccountId = "615242630409"
	//Test4AdminAccountId = "" // AWS Control Tower test
	//Test5AdminAccountId = "" // AWS IAM Identity Center test
	Test6AdminAccountId = "290222018231"
)

// Test1 returns an *awscfg.Config with the given role in the src-bin-test1
// organization's admin account.
func Test1(roleName string) *awscfg.Config {
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test1AdminAccountId,
		roleName,
		time.Hour,
	))
}

// Test2 returns an *awscfg.Config with the given role in the src-bin-test1
// organization's admin account.
func Test2(roleName string) *awscfg.Config {
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test2AdminAccountId,
		roleName,
		time.Hour,
	))
}

// Test3 returns an *awscfg.Config with the given role in the src-bin-test1
// organization's admin account.
func Test3(roleName string) *awscfg.Config {
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test3AdminAccountId,
		roleName,
		time.Hour,
	))
}

// Test4 would exist but that organization is an AWS Control Tower test
// organization and doesn't have Substrate fully bootstrapped.

// Test5 would exist but that organization is an AWS IAM Identity Center test
// organization and doesn't have Substrate fully bootstrapped.

// Test6 returns an *awscfg.Config with the given role in the src-bin-test1
// organization's admin account.
func Test6(roleName string) *awscfg.Config {
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test6AdminAccountId,
		roleName,
		time.Hour,
	))
}

// cfg returns an *awscfg.Config using whatever credentials it finds in the
// environment or blows up. If it blows up, here's how to get the appropriate
// credentials:
//
//	eval $(cd ../src-bin && substrate credentials)
//
// This won't be necessary in CodeBuild, as it's arranged to automatically
// have credentials available.
func cfg(ctx context.Context) *awscfg.Config {
	return awscfg.Must(awscfg.NewConfig(ctx)).Regional(
		"us-east-1", // the src-bin organization's default region
	)
}

// ctx returns a context structurally identical to the one created in
// cmd/substrate/main.go that's passed to awscfg.NewConfig and all the other
// interesting parts of Substrate.
func ctx() context.Context {
	return context.WithValue(
		context.WithValue(
			context.WithValue(
				context.Background(),
				"Command",
				"test",
			),
			"Subcommand",
			"test",
		),
		"Username",
		"test",
	)
}
