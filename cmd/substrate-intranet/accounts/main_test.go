package accounts

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/authorizerutil"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/cmd/substrate/role/create"
	"github.com/src-bin/substrate/cmd/substrate/role/delete"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
)

func TestAccountsConsole12hAdministrator(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	resp, err := Main(ctx, cfg, nil /* oc */, request(
		roles.Administrator,       // start as Administrator in the admin account
		roles.DeployAdministrator, // assume DeployAdministrator in the deploy account
	))
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(jsonutil.MustString(resp))
	if strings.Contains(resp.Body, awscfg.AccessDenied) {
		t.Fatal(resp.Body)
	}
	expiry, err := time.Parse(time.RFC3339, resp.Headers["X-Substrate-Credentials-Expire"])
	if err != nil {
		t.Fatal(err)
	}
	if duration := expiry.Sub(time.Now()); duration < 11*time.Hour {
		t.Fatal(duration)
	}
}

func TestAccountsConsole12hDeveloper(t *testing.T) {
	const roleName = "TestDeveloper"
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()

	cmdutil.OverrideArgs(
		create.Command(),
		"--role", roleName,
		"--humans",
		"--special", accounts.Deploy,
		"--administrator-access",
	)
	create.Main(ctx, cfg, nil, nil, os.Stdout)
	defer func() {
		cmdutil.OverrideArgs(delete.Command(), "--force", "--role", roleName)
		delete.Main(ctx, cfg, nil, nil, os.Stdout)
		time.Sleep(10 * time.Second) // give AWS IAM time to sort itself out
	}()
	time.Sleep(10 * time.Second) // give AWS IAM time to sort itself out

	resp, err := Main(ctx, cfg, nil /* oc */, request(
		roleName, // start as TestDeveloper in the admin account
		roleName, // assume TestDeveloper in the deploy account
	))
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(jsonutil.MustString(resp))
	if strings.Contains(resp.Body, awscfg.AccessDenied) {
		t.Fatal(resp.Body)
	}
	expiry, err := time.Parse(time.RFC3339, resp.Headers["X-Substrate-Credentials-Expire"])
	if err != nil {
		t.Fatal(err)
	}
	if duration := expiry.Sub(time.Now()); duration < 11*time.Hour {
		t.Fatal(duration)
	}
}

func TestAccountsConsoleDenied(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	resp, err := Main(ctx, cfg, nil /* oc */, request(
		roles.Auditor,             // start as Auditor in the admin account
		roles.DeployAdministrator, // assume DeployAdministrator in the deploy account, which will fail
	))
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(jsonutil.MustString(resp))
	if !strings.Contains(resp.Body, awscfg.AccessDenied) {
		t.Fatal(resp.Body)
	}
}

func request(initialRoleName, finalRoleName string) *events.APIGatewayV2HTTPRequest {
	return &events.APIGatewayV2HTTPRequest{
		QueryStringParameters: map[string]string{
			"number": "903998760555", // test1 deploy account
			"role":   finalRoleName,
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			AccountID: testawscfg.Test1SubstrateAccountId,
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				Lambda: map[string]interface{}{
					authorizerutil.RoleName: initialRoleName,
				},
			},
		},
	}
}
