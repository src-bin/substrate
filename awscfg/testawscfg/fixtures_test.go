package testawscfg

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/roles"
)

func Test0(t *testing.T) {
	ctx := context.Background()
	if _, err := awscfg.Must(awscfg.NewConfig(ctx)).Regional("us-east-1").GetCallerIdentity(ctx); err != nil {
		t.Fatal(err)
		fmt.Printf(
			"no AWS credentials; run `eval $(cd ../src-bin && substrate credentials)` to get some",
		)
		os.Exit(1) // not t.FailNow() so that it aborts the whole test program if it fails
	}
}

func Test1Administrator(t *testing.T) {
	testFixture(t, Test1, Test1SubstrateAccountId, roles.Administrator)
}

func Test1Auditor(t *testing.T) {
	testFixture(t, Test1, Test1SubstrateAccountId, roles.Auditor)
}

func Test2Administrator(t *testing.T) {
	testFixture(t, Test2, Test2SubstrateAccountId, roles.Administrator)
}

func Test2Auditor(t *testing.T) {
	testFixture(t, Test2, Test2SubstrateAccountId, roles.Auditor)
}

func Test3Administrator(t *testing.T) {
	testFixture(t, Test3, Test3SubstrateAccountId, roles.Administrator)
}

func Test3Auditor(t *testing.T) {
	testFixture(t, Test3, Test3SubstrateAccountId, roles.Auditor)
}

func Test4Administrator(t *testing.T) {
	testFixture(t, Test4, Test4SubstrateAccountId, roles.Administrator)
}

func Test4Auditor(t *testing.T) {
	testFixture(t, Test4, Test4SubstrateAccountId, roles.Auditor)
}

func Test6Administrator(t *testing.T) {
	testFixture(t, Test6, Test6SubstrateAccountId, roles.Administrator)
}

func Test6Auditor(t *testing.T) {
	testFixture(t, Test6, Test6SubstrateAccountId, roles.Auditor)
}

// These haven't had src-bin's Administrator added to their
// substrate.Administrator.assume-role-policy.json yet.
/*
func Test7Administrator(t *testing.T) {
	testFixture(t, Test7, Test7SubstrateAccountId, roles.Administrator)
}

func Test7Auditor(t *testing.T) {
	testFixture(t, Test7, Test7SubstrateAccountId, roles.Auditor)
}

func Test8Administrator(t *testing.T) {
	testFixture(t, Test8, Test8SubstrateAccountId, roles.Administrator)
}

func Test8Auditor(t *testing.T) {
	testFixture(t, Test8, Test8SubstrateAccountId, roles.Auditor)
}

func Test9Administrator(t *testing.T) {
	testFixture(t, Test9, Test9SubstrateAccountId, roles.Administrator)
}

func Test9Auditor(t *testing.T) {
	testFixture(t, Test9, Test9SubstrateAccountId, roles.Auditor)
}

func Test10Administrator(t *testing.T) {
	testFixture(t, Test10, Test10SubstrateAccountId, roles.Administrator)
}

func Test10Auditor(t *testing.T) {
	testFixture(t, Test10, Test10SubstrateAccountId, roles.Auditor)
}

func Test11Administrator(t *testing.T) {
	testFixture(t, Test11, Test11SubstrateAccountId, roles.Administrator)
}

func Test11Auditor(t *testing.T) {
	testFixture(t, Test11, Test11SubstrateAccountId, roles.Auditor)
}
*/

func testFixture(
	t *testing.T,
	f func(string) (*awscfg.Config, func()),
	accountId, roleName string,
) {
	ctx := context.Background()
	cfg, restore := f(roleName)
	defer restore()
	callerIdentity, err := cfg.GetCallerIdentity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if aws.ToString(callerIdentity.Arn) != fmt.Sprintf(
		"arn:aws:sts::%s:assumed-role/%s/test",
		accountId,
		roleName,
	) {
		t.Fatal(jsonutil.MustString(callerIdentity))
	}
}
