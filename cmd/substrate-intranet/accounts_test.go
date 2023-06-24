package main

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
	createrole "github.com/src-bin/substrate/cmd/substrate/create-role"
	deleterole "github.com/src-bin/substrate/cmd/substrate/delete-role"
	"github.com/src-bin/substrate/cmdutil"
	"github.com/src-bin/substrate/roles"
)

func TestAccountsConsole12hAdministrator(t *testing.T) {
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	resp, err := accountsHandler(ctx, cfg, nil /* oc */, apiGatewayProxyRequest(
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
	if duration := expiry.Sub(time.Now()); duration < 59*time.Minute /* 11*time.Hour */ {
		t.Fatal(duration)
	}
}

func TestAccountsConsole12hDeveloper(t *testing.T) {
	const roleName = "TestDeveloper"
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)

	cmdutil.OverrideArgs("-role", roleName, "-humans", "-special", accounts.Deploy, "-administrator-access")
	createrole.Main(ctx, cfg, os.Stdout)
	defer func() {
		cmdutil.OverrideArgs("-delete", "-role", roleName)
		deleterole.Main(ctx, cfg, os.Stdout)
	}()

	resp, err := accountsHandler(ctx, cfg, nil /* oc */, apiGatewayProxyRequest(
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
	if duration := expiry.Sub(time.Now()); duration < 59*time.Minute /* 11*time.Hour */ {
		t.Fatal(duration)
	}
}

func TestAccountsConsoleDenied(t *testing.T) {
	ctx := context.Background()
	cfg := testawscfg.Test1(roles.Administrator)
	resp, err := accountsHandler(ctx, cfg, nil /* oc */, apiGatewayProxyRequest(
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

func apiGatewayProxyRequest(initialRoleName, finalRoleName string) *events.APIGatewayProxyRequest {
	return &events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"number": "903998760555", // test1 deploy account
			"role":   finalRoleName,
		},
		RequestContext: events.APIGatewayProxyRequestContext{
			AccountID: testawscfg.Test1AdminAccountId,
			Authorizer: map[string]interface{}{
				authorizerutil.RoleName: initialRoleName,
			},
		},
	}
}
