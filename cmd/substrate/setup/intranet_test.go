package setup

import (
	"context"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/roles"
)

func TestIntranet2Test1(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	dnsDomainName, idpName := intranet(ctx, mgmtCfg, substrateCfg)
	if dnsDomainName != "src-bin-test1.net" {
		t.Fatalf(`%q != "src-bin-test1.net"`, dnsDomainName)
	}
	if idpName != oauthoidc.Okta {
		t.Fatalf(`%q != %q`, idpName, oauthoidc.Okta)
	}
	t.Log(dnsDomainName, idpName)
}

func TestIntranet2Test2(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test2(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	dnsDomainName, idpName := intranet(ctx, mgmtCfg, substrateCfg)
	if dnsDomainName != "src-bin-test2.net" {
		t.Fatalf(`%q != "src-bin-test2.net"`, dnsDomainName)
	}
	if idpName != oauthoidc.Okta {
		t.Fatalf(`%q != %q`, idpName, oauthoidc.Okta)
	}
	t.Log(dnsDomainName, idpName)
}
