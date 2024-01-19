package setup

import (
	"context"
	"time"

	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

func network2(ctx context.Context, mgmtCfg *awscfg.Config) {

	// Try to assume the NetworkAdministrator role in the special network
	// account but give up without a fight if we can't since new
	// installations won't have this role or even this account and that's just
	// fine.
	cfg, err := mgmtCfg.AssumeSpecialRole(
		ctx,
		accounts.Network,
		roles.NetworkAdministrator,
		time.Hour,
	)
	if err == nil {
		ui.Print("successfully assumed the NetworkAdministrator role; proceeding with Terraform for the network account")
	} else {
		ui.Print("could not assume the NetworkAdministrator role; continuing without the network account")
		return
	}
	accountId := cfg.MustAccountId(ctx)

	// Configure the allocator for admin networks to use 192.168.0.0/16 and
	// 21-bit subnet masks which yields 2,048 IP addresses per VPC and 32
	// possible VPCs while keeping a tidy source IP address range for granting
	// SSH and other administrative access safely and easily.
	adminNetDoc, err := networks.ReadDocument(networks.AdminFilename, networks.RFC1918_192_168_0_0_16, 21)
	ui.Must(err)
	//log.Printf("%+v", adminNetDoc)

	// Configure the allocator for normal (environment, quality) networks to use
	// 10.0.0.0/8 and 18-bit subnet masks which yields 16,384 IP addresses per
	// VPC and 1,024 possible VPCs.
	netDoc, err := networks.ReadDocument(networks.Filename, networks.RFC1918_10_0_0_0_8, 18)
	ui.Must(err)
	//log.Printf("%+v", netDoc)

	veqpDoc, err := veqp.ReadDocument()
	ui.Must(err)

	// This is a little awkward to duplicate from Main but it's expedient and
	// leaves our options open for how we do NAT Gateways when we get rid of
	// all these local files eventually.
	natGateways, err := ui.ConfirmFile(
		networks.NATGatewaysFilename,
		`do you want to provision NAT Gateways for IPv4 traffic from your private subnets to the Internet? (yes/no; answering "yes" costs about $100 per month per region per environment/quality pair)`,
	)
	ui.Must(err)

	ui.Print(cfg)
	ui.Debug(accountId)
	ui.Debug(adminNetDoc)
	ui.Debug(netDoc)
	ui.Debug(veqpDoc)
	ui.Debug(natGateways)

}
