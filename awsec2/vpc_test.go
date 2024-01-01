package awsec2

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/awscfg/testawscfg"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/roles"
)

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

func TestEnsureSecurityGroup(t *testing.T) {
	ctx := context.Background()
	cfg, restore := testawscfg.Test1(roles.Administrator)
	defer restore()
	cfg = cfg.Regional("us-west-2")
	vpcs, err := DescribeVPCs(ctx, cfg, naming.Admin, naming.Default)
	if err != nil {
		t.Fatal(err)
	}
	if len(vpcs) != 1 {
		t.Fatal(vpcs)
	}
	securityGroup, err := EnsureSecurityGroup(ctx, cfg, aws.ToString(vpcs[0].VpcId), naming.InstanceFactory, []int{22})
	if err != nil {
		t.Fatal(err)
	}
	_ = securityGroup
	// t.Log(jsonutil.MustString(securityGroup))
}
