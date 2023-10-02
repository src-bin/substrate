package setup

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/oauthoidc"
	"github.com/src-bin/substrate/roles"
)

func TestIntranet2Test1(t *testing.T) {
	ctx := context.Background()
	substrateCfg := testawscfg.Test1(roles.Administrator)
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	dnsDomainName2, idpName2 := intranet2(ctx, mgmtCfg, substrateCfg)
	/*
		if dnsDomainName2 != "src-bin-test1.net" {
			t.Fatalf(`%q != "src-bin-test1.net"`, dnsDomainName2)
		}
	*/
	if !strings.HasSuffix(dnsDomainName2, ".cloudfront.net") {
		t.Fatalf(`%q does not end with ".cloudfront.net"`, dnsDomainName2)
	}
	if idpName2 != oauthoidc.Okta {
		t.Fatalf(`%q != %q`, idpName2, oauthoidc.Okta)
	}
	t.Log(dnsDomainName2, idpName2)
}

func TestIntranet2Test2(t *testing.T) {
	ctx := context.Background()
	substrateCfg := testawscfg.Test2(roles.Administrator)
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	dnsDomainName2, idpName2 := intranet2(ctx, mgmtCfg, substrateCfg)
	/*
		if dnsDomainName2 != "src-bin-test2.net" {
			t.Fatalf(`%q != "src-bin-test2.net"`, dnsDomainName2)
		}
	*/
	if !strings.HasSuffix(dnsDomainName2, ".cloudfront.net") {
		t.Fatalf(`%q does not end with ".cloudfront.net"`, dnsDomainName2)
	}
	if idpName2 != oauthoidc.Okta {
		t.Fatalf(`%q != %q`, idpName2, oauthoidc.Okta)
	}
	t.Log(dnsDomainName2, idpName2)
}
