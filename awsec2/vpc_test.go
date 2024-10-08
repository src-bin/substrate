package awsec2

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/cidr"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
)

func TestCreateDeleteVPC(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	cfg = awscfg.Must(cfg.AssumeSpecialRole(ctx, naming.Network, roles.NetworkAdministrator, time.Hour)).Regional("us-west-2")

	vpcs, err := DescribeVPCs(ctx, cfg, "test", "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, vpc := range vpcs {
		if err := DeleteVPC(ctx, cfg, aws.ToString(vpc.VpcId)); err != nil {
			t.Fatal(err)
		}
	}

	vpc, err := EnsureVPC(ctx, cfg, "test", "test", ui.Must2(cidr.ParseIPv4("10.0.0.0/16")), nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = vpc
	//t.Log(jsonutil.MustString(vpc))

	if err := DeleteVPC(ctx, cfg, aws.ToString(vpc.VpcId)); err != nil {
		t.Fatal(err)
	}
}

func TestDescribeVPCs(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	cfg = cfg.Regional("us-west-2")

	// We don't need to switch into the network account, which saves a little
	// time, since we know this network's already shared and tagged.
	//cfg = awscfg.Must(cfg.AssumeSpecialRole(ctx, accounts.Network, roles.Auditor, time.Hour)).Regional("us-west-2")

	vpcs, err := DescribeVPCs(ctx, cfg, naming.Admin, naming.Default)
	if err != nil {
		t.Fatal(err)
	}
	if len(vpcs) != 1 {
		t.Fatal(vpcs)
	}
}

func TestEnsureVPC(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	cfg = awscfg.Must(cfg.AssumeSpecialRole(ctx, naming.Network, roles.NetworkAdministrator, time.Hour)).Regional("us-west-2")

	vpc, err := EnsureVPC(ctx, cfg, "staging", "default", ui.Must2(cidr.ParseIPv4("10.0.0.0/18")), nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = vpc
	//t.Log(jsonutil.MustString(vpc))
}
