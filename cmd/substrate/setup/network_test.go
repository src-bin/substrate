package setup

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/roles"
)

func TestNetworkTest1(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	network(ctx, mgmtCfg)
}

func TestNetworkTest2(t *testing.T) {
	ctx := context.Background()
	substrateCfg, restore := testawscfg.Test2(roles.Administrator)
	defer restore()
	mgmtCfg := awscfg.Must(substrateCfg.AssumeManagementRole(ctx, roles.Substrate, time.Hour))
	network(ctx, mgmtCfg)
}

func init() {
	autoApprove = aws.Bool(false)
	ignoreServiceQuotas = aws.Bool(true)
	noApply = aws.Bool(true)
}
