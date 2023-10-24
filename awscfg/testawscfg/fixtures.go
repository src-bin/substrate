package testawscfg

import (
	"context"
	"os"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/contextutil"
)

const (
	Test1SubstrateAccountId = "716893237583"
	Test2SubstrateAccountId = "944106955638"
	Test3SubstrateAccountId = "615242630409"
	Test4SubstrateAccountId = "581144495976" // initially an AWS Control Tower test
	//Test5SubstrateAccountId = "" // AWS IAM Identity Center test
	Test6SubstrateAccountId  = "290222018231"
	Test7SubstrateAccountId  = "119320875853"
	Test8SubstrateAccountId  = "283931283135"
	Test9SubstrateAccountId  = "981340593605"
	Test10SubstrateAccountId = "158812816352"
	Test11SubstrateAccountId = "904600466829"
)

// Test1 returns an *awscfg.Config with the given role in the src-bin-test1
// organization's Substrate account.
func Test1(roleName string) *awscfg.Config {
	substrateRoot("test1")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test1SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test2 returns an *awscfg.Config with the given role in the src-bin-test2
// organization's Substrate account.
func Test2(roleName string) *awscfg.Config {
	substrateRoot("test2")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test2SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test3 returns an *awscfg.Config with the given role in the src-bin-test3
// organization's Substrate account.
func Test3(roleName string) *awscfg.Config {
	substrateRoot("test3")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test3SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test4 returns an *awscfg.Config with the given role in the src-bin-test4
// organization's Substrate account.
func Test4(roleName string) *awscfg.Config {
	substrateRoot("test4")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test4SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test5 would exist but that organization is an AWS IAM Identity Center test
// organization and doesn't have Substrate fully bootstrapped.

// Test6 returns an *awscfg.Config with the given role in the src-bin-test6
// organization's Substrate account.
func Test6(roleName string) *awscfg.Config {
	substrateRoot("test6")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test6SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test7 returns an *awscfg.Config with the given role in the src-bin-test7
// organization's Substrate account.
func Test7(roleName string) *awscfg.Config {
	substrateRoot("test7")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test7SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test8 returns an *awscfg.Config with the given role in the src-bin-test8
// organization's Substrate account.
func Test8(roleName string) *awscfg.Config {
	substrateRoot("test8")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test8SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test9 returns an *awscfg.Config with the given role in the src-bin-test9
// organization's Substrate account.
func Test9(roleName string) *awscfg.Config {
	substrateRoot("test9")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test9SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test10 returns an *awscfg.Config with the given role in the src-bin-test10
// organization's Substrate account.
func Test10(roleName string) *awscfg.Config {
	substrateRoot("test10")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test10SubstrateAccountId,
		roleName,
		time.Hour,
	))
}

// Test11 returns an *awscfg.Config with the given role in the src-bin-test11
// organization's Substrate account.
func Test11(roleName string) *awscfg.Config {
	substrateRoot("test11")
	return awscfg.Must(cfg(ctx()).AssumeRole(
		ctx(),
		Test11SubstrateAccountId,
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
				contextutil.Command,
				"test",
			),
			contextutil.Subcommand,
			"test",
		),
		contextutil.Username,
		"test",
	)
}

func substrateRoot(repo string) {
	for {
		if err := os.Chdir(repo); err == nil {
			break
		}
		if dirname, err := os.Getwd(); err == nil && dirname == "/" {
			break // panic(fmt.Sprintf("%s not found in any parent directory"))
		}
		if err := os.Chdir(".."); err != nil {
			panic(err)
		}
	}
}
