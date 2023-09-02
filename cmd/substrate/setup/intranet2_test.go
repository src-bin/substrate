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

func TestIntranet2(t *testing.T) {
	t.Skip()
	ctx := context.Background()
	substrateCfg := testawscfg.Test1(roles.Administrator)
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	dnsDomainName2, idpName2 := intranet2(ctx, mgmtCfg, substrateCfg)
	if dnsDomainName2 != "src-bin-test1.net" {
		t.Fatalf(`%q != "src-bin-test1.net"`, dnsDomainName2)
	}
	if idpName2 != oauthoidc.Okta {
		t.Fatalf(`%q != %q`, idpName2, oauthoidc.Okta)
	}
}
