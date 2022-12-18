package testawscfg

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/roles"
)

func Test0(t *testing.T) {
	_, err := cfg(ctx()).GetCallerIdentity(ctx())
	if err != nil {
		fmt.Printf(
			"no AWS credentials; run `eval $(cd ../src-bin && substrate credentials)` to get some",
		)
		os.Exit(1) // not t.FailNow() so that it aborts the whole test program if it fails
	}
}

func Test1Administrator(t *testing.T) {
	testFixture(t, Test1, Test1AdminAccountId, roles.Administrator)
}

func Test1Auditor(t *testing.T) {
	testFixture(t, Test1, Test1AdminAccountId, roles.Auditor)
}

func Test2Administrator(t *testing.T) {
	testFixture(t, Test2, Test2AdminAccountId, roles.Administrator)
}

func Test2Auditor(t *testing.T) {
	testFixture(t, Test2, Test2AdminAccountId, roles.Auditor)
}

func Test3Administrator(t *testing.T) {
	testFixture(t, Test3, Test3AdminAccountId, roles.Administrator)
}

func Test3Auditor(t *testing.T) {
	testFixture(t, Test3, Test3AdminAccountId, roles.Auditor)
}

func Test6Administrator(t *testing.T) {
	testFixture(t, Test6, Test6AdminAccountId, roles.Administrator)
}

func Test6Auditor(t *testing.T) {
	testFixture(t, Test6, Test6AdminAccountId, roles.Auditor)
}

func testFixture(
	t *testing.T,
	f func(string) *awscfg.Config,
	accountId, roleName string,
) {
	callerIdentity, err := f(roleName).GetCallerIdentity(ctx())
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
