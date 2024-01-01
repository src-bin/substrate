package testawscfg

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/contextutil"
	"github.com/src-bin/substrate/ui"
)

// Test*AccountId give names to a bunch of otherwise inscrutable 12-digit
// account numbers. Mostly they're used to construct the Test* functions
// below that cross from the src-bin organization to the test organizations.
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

// Test* are each a func(string) (*awscfg.Config, func()) that accepts an IAM
// role name, typically Administrator or Auditor, that exists in the
// destination organization's Substrate account. The returned *awscfg.Config
// will have assumed that role name in the test organization's Substrate
// account. Calling one of these functions overrides the AWS_ACCESS_KEY_ID,
// AWS_SECRET_ACCESS_KEY, and AWS_SESSION_TOKEN environment variables. The
// returned func() restores those environment variables to the values they had
// before calling this function; it is meant to be used with defer.
var (
	Test1 = fixture(Test1SubstrateAccountId, "test1")
	Test2 = fixture(Test2SubstrateAccountId, "test2")
	Test3 = fixture(Test3SubstrateAccountId, "test3")
	Test4 = fixture(Test4SubstrateAccountId, "test4")
	// Test5 would exist but it's an AWS IAM Identity Center test organization that doesn't have Substrate fully bootstrapped
	Test6  = fixture(Test6SubstrateAccountId, "test6")
	Test7  = fixture(Test7SubstrateAccountId, "test7")
	Test8  = fixture(Test8SubstrateAccountId, "test8")
	Test9  = fixture(Test9SubstrateAccountId, "test9")
	Test10 = fixture(Test10SubstrateAccountId, "test10")
	Test11 = fixture(Test11SubstrateAccountId, "test11")
)

func fixture(accountId, repo string) func(string) (*awscfg.Config, func()) {
	return func(roleName string) (*awscfg.Config, func()) {

		// Use a context structurally identical to the one created in
		// cmd/substrate/main.go that's passed to awscfg.NewConfig and
		// all the other interesting parts of Substrate.
		ctx := context.WithValue(
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

		// AWS config in the src-bin account, which we'll use below to assume
		// a role in the test account. This must happen before we change
		// directories or we'll trip over the the credentials' and the
		// directory's idea of the management account number not matching.
		cfg := awscfg.Must(awscfg.NewConfig(ctx)).Regional(
			"us-east-1", // the src-bin organization's default region, so IAM will have someplace to go
		)

		// Find the repo in the closest possible ancestor directory.
		oldDirname, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		for {
			if err := os.Chdir(repo); err == nil {
				dirname, err := os.Getwd()
				if err != nil {
					panic(err)
				}
				ui.Printf("testawscfg fixture: cd %s", dirname)
				break
			}
			if dirname, err := os.Getwd(); err == nil && dirname == "/" {
				break // panic(fmt.Sprintf("%s not found in any parent directory"))
			}
			if err := os.Chdir(".."); err != nil {
				panic(err)
			}
		}

		// Construct an AWS config that crosses from the src-bin organization
		// into this test organization.
		cfg = awscfg.Must(cfg.AssumeRole(
			ctx,
			accountId,
			roleName,
			time.Hour,
		))
		//log.Print(jsonutil.MustString(cfg.MustGetCallerIdentity(ctx)))

		// Take note of the credentials currently in this process' environment
		// before replacing them with credentials from the AWS config we just
		// constructed in the test organization.
		oldCreds := aws.Credentials{
			AccessKeyID:     os.Getenv(awscfg.AWS_ACCESS_KEY_ID),
			SecretAccessKey: os.Getenv(awscfg.AWS_SECRET_ACCESS_KEY),
			SessionToken:    os.Getenv(awscfg.AWS_SESSION_TOKEN),
		}
		creds, err := cfg.Retrieve(ctx)
		if err != nil {
			panic(err)
		}
		if err := awscfg.Setenv(creds); err != nil {
			panic(err)
		}

		return cfg, func() {
			if err := awscfg.Setenv(oldCreds); err != nil {
				panic(err)
			}
			if err := os.Chdir(oldDirname); err != nil {
				panic(err)
			}
			ui.Printf("testawscfg fixture: cd %s", oldDirname)
		}
	}
}
