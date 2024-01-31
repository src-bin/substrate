package awsec2

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
)

func TestInternetGateway(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	cfg = awscfg.Must(cfg.AssumeSpecialRole(ctx, naming.Network, roles.NetworkAdministrator, time.Hour)).Regional("us-west-2")

	vpcs, err := DescribeVPCs(ctx, cfg, "staging", "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(vpcs) != 1 {
		t.Fatal(vpcs)
	}
	vpcId := aws.ToString(vpcs[0].VpcId)

	if _, err := EnsureInternetGateway(ctx, cfg, vpcId); err != nil {
		t.Fatal(err)
	}
}
