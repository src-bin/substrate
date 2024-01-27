package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/availabilityzones"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsec2"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/cidr"
	"github.com/src-bin/substrate/fileutil"
	"github.com/src-bin/substrate/jsonutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/tagging"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

func ensureVPC(
	ctx context.Context,
	cfg *awscfg.Config,
	environment, quality string,
	cidrPrefixIPv4 cidr.IPv4,
	natGateways bool,
) {

	ui.Spinf("finding or creating the %s %s VPC in %s", environment, quality, cfg.Region())
	vpc, err := awsec2.EnsureVPC(ctx, cfg, environment, quality, cidrPrefixIPv4, nil)
	ui.Must(err)
	//ui.Debug(vpc)
	vpcId := aws.ToString(vpc.VpcId)
	cidrPrefixIPv6 := cidr.MustIPv6(cidr.ParseIPv6(aws.ToString(vpc.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)))
	ui.Stopf("%s %s %s", vpcId, cidrPrefixIPv4, cidrPrefixIPv6)

	azs, err := availabilityzones.Select(ctx, cfg, cfg.Region(), availabilityzones.NumberPerNetwork)
	ui.Must(err)
	ui.Printf("using availability zones %s", strings.Join(azs, ", "))

	// Decide how many additional bits to use for each public subnet's CIDR
	// prefix. Private subnets, if a network has them, always use two. The
	// layout when there are private subnets is as follows:
	//
	//   |    -    | public | public | public |
	//   |               private              |
	//   |               private              |
	//   |               private              |
	//
	// If the CIDR prefix length is 18 then this results in three public /22
	// subnets and three private /20 subnets. The very first /22 is wasted. If
	// there are no private subnets then the public subnets are /20 and the
	// first /20 is wasted.
	bits := 2
	hasPrivateSubnets := environment != "admin"
	if hasPrivateSubnets {
		bits = 4
	}

	// TODO InternetGateway
	if hasPrivateSubnets {
		// TODO EgressOnlyInternetGateway
	}

	// Three public and maybe three private subnets, too. One wasted subnet
	// at the beginning.
	publicRouteTable, privateRouteTables, err := awsec2.DescribeRouteTables(ctx, cfg, vpcId)
	ui.Must(err)
	for i, az := range azs {

		ui.Spinf("finding or creating a public subnet in %s", az)
		publicSubnet, err := awsec2.EnsureSubnet(
			ctx,
			cfg,
			vpcId,
			az,
			cidr.MustIPv4(cidrPrefixIPv4.SubnetIPv4(bits, i+1)),
			cidr.MustIPv6(cidrPrefixIPv6.SubnetIPv6(8, i+1)),
			tagging.Map{
				tagging.Connectivity: "public",
				tagging.Environment:  environment,
				tagging.Name:         fmt.Sprintf("%s-%s-public-%s", environment, quality, az),
				tagging.Quality:      quality,
			},
		)
		ui.Must(err)
		ui.Stopf("%s %s %s", publicSubnet.SubnetId, publicSubnet.CidrBlock, publicSubnet.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)
		//ui.Debug(publicSubnet)

		// TODO routes to InternetGateway

		if hasPrivateSubnets {
			ui.Spinf("finding or creating a private subnet in %s", az)
			privateSubnet, err := awsec2.EnsureSubnet(
				ctx,
				cfg,
				vpcId,
				az,
				cidr.MustIPv4(cidrPrefixIPv4.SubnetIPv4(2, i+1)),
				cidr.MustIPv6(cidrPrefixIPv6.SubnetIPv6(8, i+0x81)), // to shift past the one wasted and three public subnets
				tagging.Map{
					tagging.Connectivity: "private",
					tagging.Environment:  environment,
					tagging.Name:         fmt.Sprintf("%s-%s-private-%s", environment, quality, az),
					tagging.Quality:      quality,
				},
			)
			ui.Must(err)
			// TODO maybe CreateRouteTable AssociateRouteTable
			ui.Stopf("%s %s %s", privateSubnet.SubnetId, privateSubnet.CidrBlock, privateSubnet.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)
			//ui.Debug(privateSubnet)

			if natGateways {
				// TODO maybe an EIP and NAT Gateway in publicSubnet with a route to it here
			}

			// TODO route to EgressOnlyInternetGateway
		}

	}
	publicRouteTable, privateRouteTables, err = awsec2.DescribeRouteTables(ctx, cfg, vpcId)
	ui.Must(err)

	// TODO VPC Endpoints for DynamoDB and S3 with all route tables

	/////////////////////////////////////////////////////////////

	/*

	   // IPv4 and IPv6 Internet Gateways for the public subnets.

	   	igw := terraform.InternetGateway{
	   		Label: terraform.Label(vpc.Tags),
	   		Tags:  vpc.Tags,
	   		VpcId: terraform.U(vpc.Ref(), ".id"),
	   	}

	   ui.Must(terraform.StateRm(dirname, igw.Ref().Value()))

	   	ui.Must(terraform.StateRm(dirname, terraform.Route{
	   		DestinationIPv4:   terraform.Q("0.0.0.0/0"),
	   		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
	   		Label:             terraform.Label(vpc.Tags, "public-internet-ipv4"),
	   		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	   	}.Ref().Value()))

	   	ui.Must(terraform.StateRm(dirname, terraform.Route{
	   		DestinationIPv6:   terraform.Q("::/0"),
	   		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
	   		Label:             terraform.Label(vpc.Tags, "public-internet-ipv6"),
	   		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	   	}.Ref().Value()))

	   // IPv6 Egress-Only Internet Gateway for the private subnets.  (The IPv4
	   // NAT Gateway comes later because it's per-subnet.  That is also where
	   // this Egress-Only Internat Gateway is associated with the route table.)

	   	egw := terraform.EgressOnlyInternetGateway{
	   		Label: terraform.Label(vpc.Tags),
	   		Tags:  vpc.Tags,
	   		VpcId: terraform.U(vpc.Ref(), ".id"),
	   	}

	   	if hasPrivateSubnets {
	   		ui.Must(terraform.StateRm(dirname, egw.Ref().Value()))
	   	}

	   // VPC Endpoint for S3, the one VPC Endpoint everyone's all but guaranteed to need.

	   	vpce := terraform.VPCEndpoint{
	   		Label: terraform.Label(vpc.Tags),
	   		RouteTableIds: terraform.ValueSlice{
	   			terraform.U(vpc.Ref(), ".default_route_table_id"),
	   			// more will be appeneded before this function returns
	   		},
	   		ServiceName: terraform.Qf("com.amazonaws.%s.s3", region),
	   		Tags:        vpc.Tags,
	   		VpcId:       terraform.U(vpc.Ref(), ".id"),
	   	}

	   	for i, az := range azs {

	   		// Explicitly associate the public subnets with the main routing table.
	   		ui.Must(terraform.StateRm(dirname, terraform.RouteTableAssociation{
	   			Label:        s.Label,
	   			RouteTableId: terraform.U(vpc.Ref(), ".default_route_table_id"),
	   			SubnetId:     terraform.U(s.Ref(), ".id"),
	   		}.Ref().Value()))

	   		if !hasPrivateSubnets {
	   			continue
	   		}

	   		// Save a reference to the public subnet in this availability zone
	   		// so we know where to put the NAT Gateway.
	   		natGatewaySubnetId := terraform.U(s.Ref(), ".id")

	   		// Private subnets need their own routing tables to keep their NAT
	   		// Gateway traffic zonal.  The VPC Endpoint we created for S3 needs
	   		// to be made aware of this routing table, too.
	   		rt := terraform.RouteTable{
	   			Label: s.Label,
	   			Tags:  s.Tags,
	   			VpcId: terraform.U(vpc.Ref(), ".id"),
	   		}
	   		ui.Must(terraform.StateRm(dirname, rt.Ref().Value()))
	   		ui.Must(terraform.StateRm(dirname, terraform.RouteTableAssociation{
	   			Label:        s.Label,
	   			RouteTableId: terraform.U(rt.Ref(), ".id"),
	   			SubnetId:     terraform.U(s.Ref(), ".id"),
	   		}.Ref().Value()))
	   		vpce.RouteTableIds = append(vpce.RouteTableIds, terraform.U(rt.Ref(), ".id"))

	   		// NAT Gateway for this private subnet.
	   		eip := terraform.EIP{
	   			Commented:          !natGateways,
	   			InternetGatewayRef: igw.Ref(),
	   			Label:              terraform.Label(tags),
	   			Tags:               tags,
	   		}
	   		eip.Tags.Name = vpc.Tags.Name + "-nat-gateway-" + az
	   		ui.Must(terraform.StateRm(dirname, eip.Ref().Value()))
	   		ngw := terraform.NATGateway{
	   			Commented: !natGateways,
	   			Label:     terraform.Label(tags),
	   			SubnetId:  natGatewaySubnetId,
	   			Tags:      tags,
	   		}
	   		ngw.Tags.Name = vpc.Tags.Name + "-" + az
	   		if natGateways {
	   			ui.Must(terraform.StateRm(dirname, ngw.Ref().Value()))
	   			ui.Must(terraform.StateRm(dirname, terraform.Route{
	   				Commented:       !natGateways,
	   				DestinationIPv4: terraform.Q("0.0.0.0/0"),
	   				Label:           terraform.Label(tags),
	   				NATGatewayId:    terraform.U(ngw.Ref(), ".id"),
	   				RouteTableId:    terraform.U(rt.Ref(), ".id"),
	   			}.Ref().Value()))
	   		}

	   		// Associate the VPC's Egress-Only Internet Gateway for IPv6 traffic.
	   		ui.Must(terraform.StateRm(dirname, terraform.Route{
	   			DestinationIPv6:             terraform.Q("::/0"),
	   			EgressOnlyInternetGatewayId: terraform.U(egw.Ref(), ".id"),
	   			Label:                       terraform.Label(s.Tags, "private-internet-ipv6"),
	   			RouteTableId:                terraform.U(rt.Ref(), ".id"),
	   		}.Ref().Value()))

	   }

	   // Now that all the route tables have been associated with the S3 VPC
	   // Endpoint, add it to the file.
	   ui.Must(terraform.StateRm(dirname, vpce.Ref().Value()))
	*/
}

func network(ctx context.Context, mgmtCfg *awscfg.Config) {

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

	// This is a little awkward to duplicate from Main but it's expedient and
	// leaves our options open for how we do NAT Gateways when we get rid of
	// all these local files eventually.
	natGateways, err := ui.ConfirmFile(
		networks.NATGatewaysFilename,
		`do you want to provision NAT Gateways for IPv4 traffic from your private subnets to the Internet? (yes/no; answering "yes" costs about $100 per month per region per environment/quality pair)`,
	)
	ui.Must(err)

	// Assign CIDR prefixes to the various networks.
	//
	// Substrate (formerly admin) networks are allocated from 192.168.0.0/16
	// and use 21-bit subnet masks which yields 2,048 IP addresses per VPC and
	// 32 possible VPCs while keeping a tidy source IP address range for
	// granting SSH and other administrative access safely and easily.
	//
	// Service account (environment, quality) networks are allocated from
	// 10.0.0.0/8 and use 18-bit subnet masks which yields 16,384 IP addresses
	// per VPC and 1,024 possible VPCs.
	ui.Printf("configuring networks for every environment and quality in %d regions", len(regions.Selected()))
	adminNetDoc, err := networks.ReadDocument(networks.AdminFilename, cidr.RFC1918_192_168_0_0_16, 21)
	ui.Must(err)
	//log.Printf("%+v", adminNetDoc)
	netDoc, err := networks.ReadDocument(networks.Filename, cidr.RFC1918_10_0_0_0_8, 18)
	ui.Must(err)
	//log.Printf("%+v", netDoc)
	veqpDoc, err := veqp.ReadDocument()
	ui.Must(err)
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region := range regions.Selected() {
			ui.Spinf(
				"finding or assigning an IP address range to the %s %s network in %s",
				eq.Environment,
				eq.Quality,
				region,
			)
			var doc *networks.Document
			if eq.Environment == "admin" {
				doc = adminNetDoc
			} else {
				doc = netDoc
			}
			n, err := doc.Ensure(&networks.Network{
				// TODO maybe support an alternative tagging regime for the Instance Factory's VPC
				Environment: eq.Environment,
				Quality:     eq.Quality,
				Region:      region,
			})
			ui.Must(err)
			ui.Stop(n.IPv4)
		}
	}

	// Write to substrate.admin-networks.json and substrate.networks.json once
	// more so that, even if no changes were made, formatting changes and
	// SubstrateVersion are changed.
	ui.Must(adminNetDoc.Write())
	ui.Must(netDoc.Write())

	// Ensure the VPCs-per-region service quota and a few others aren't going to get in the way.
	var deadline time.Time
	if *ignoreServiceQuotas {
		deadline = time.Now()
	}
	ui.Print("raising the VPC, Internet, Egress-Only Internet, and NAT Gateway, and EIP service quotas in all your regions (this could take days, unfortunately; this program is safe to re-run)")
	adminNets := len(adminNetDoc.FindAll(&networks.Network{Region: regions.Selected()[0]})) // admin networks per region
	nets := len(netDoc.FindAll(&networks.Network{Region: regions.Selected()[0]}))           // (environment, quality) pairs per region
	for _, quota := range [][2]string{
		{"L-F678F1CE", "vpc"}, // VPCs per region
		{"L-45FE3B85", "vpc"}, // Egress-Only Internet Gateways per region
		{"L-A4707A72", "vpc"}, // Internet Gateways per region
	} {
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			ctx,
			cfg,
			quota[0], quota[1],
			float64(adminNets+nets), // admin and non-admin VPCs per region, each with one of each type of Internet Gateway
			float64(adminNets+nets), // same value because they hassle us so much about raising the limit at all
			deadline,
		); err != nil {
			if _, ok := err.(awsservicequotas.DeadlinePassed); ok {
				ui.Print(err)
			} else {
				ui.Fatal(err)
			}
		}
	}
	if natGateways {
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			ctx,
			cfg,
			"L-FE5A380F", "vpc", // NAT Gateways per availability zone
			float64(nets), // only non-admin networks get private subnets and thus NAT Gateways
			float64(nets), // same value because they hassle us so much about raising the limit at all
			deadline,
		); err != nil {
			if _, ok := err.(awsservicequotas.DeadlinePassed); ok {
				ui.Print(err)
			} else {
				ui.Fatal(err)
			}
		}
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			ctx,
			cfg,
			"L-0263D0A3", "ec2", // EIPs per region
			float64(nets*availabilityzones.NumberPerNetwork), // NAT Gateways per AZ times the number of AZs per network
			float64(nets*availabilityzones.NumberPerNetwork), // same value because they hassle us so much about raising the limit at all
			deadline,
		); err != nil {
			if _, ok := err.(awsservicequotas.DeadlinePassed); ok {
				ui.Print(err)
			} else {
				ui.Fatal(err)
			}
		}
	}

	// Define networks for each environment and quality.  No peering yet as
	// it's difficult to reason about before all networks are created.
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region := range regions.Selected() {

			var doc *networks.Document
			if eq.Environment == "admin" {
				doc = adminNetDoc
			} else {
				doc = netDoc
			}
			n := doc.Find(&networks.Network{
				// TODO maybe support an alternative tagging regime for the Instance Factory's VPC
				Environment: eq.Environment,
				Quality:     eq.Quality,
				Region:      region,
			})
			if n == nil {
				ui.Fatal("couldn't find assigned CIDR prefix for %s %s in %s", eq.Environment, eq.Quality, region)
			}
			ensureVPC(ctx, cfg.Regional(region), eq.Environment, eq.Quality, n.IPv4, natGateways)

			terraformVPC(ctx, mgmtCfg, cfg, eq.Environment, eq.Quality, region, natGateways)

		}
	}

	// Now that all the networks exist, establish a fully-connected mesh of
	// peering connections within each environment's qualities and regions.
	ui.Must(fileutil.Remove(filepath.Join(terraform.ModulesDirname, "peering-connection/main.tf")))
	ui.Must(fileutil.Remove(filepath.Join(terraform.ModulesDirname, "peering-connection/variables.tf")))
	ui.Must(fileutil.Remove(filepath.Join(terraform.ModulesDirname, "peering-connection/versions.tf")))
	if err := fileutil.Remove(filepath.Join(terraform.ModulesDirname, "peering-connection")); err != nil {
		ui.Printf(
			"warning: failed to remove %s, which should now be empty (%s)",
			filepath.Join(terraform.ModulesDirname, "peering-connection"),
			err,
		)
	}
	networkCfg := awscfg.Must(mgmtCfg.AssumeSpecialRole(ctx, accounts.Network, roles.NetworkAdministrator, time.Hour))
	/*
		for _, region := range regions.Selected() {
			ui.Debug(awsec2.DescribeVPCPeeringConnections(ctx, networkCfg.Regional(region)))
		}
	*/
	peeringConnections, err := networks.EnumeratePeeringConnections()
	ui.Must(err)
	for _, pc := range peeringConnections.Slice() {
		eq0, eq1, region0, region1 := pc.Ends()

		ui.Spinf(
			"configuring VPC peering between %s %s %s and %s %s %s",
			eq0.Environment, eq0.Quality, region0,
			eq1.Environment, eq1.Quality, region1,
		)

		dirname := filepath.Join(
			terraform.RootModulesDirname,
			accounts.Network,
			"peering",
			eq0.Environment,
			eq1.Environment,
			eq0.Quality,
			eq1.Quality,
			region0,
			region1,
		)
		ui.Must(fileutil.Remove(filepath.Join(dirname, ".gitignore")))
		ui.Must(fileutil.Remove(filepath.Join(dirname, "main.tf")))
		ui.Must(fileutil.Remove(filepath.Join(dirname, "Makefile")))
		ui.Must(fileutil.Remove(filepath.Join(dirname, "providers.tf")))
		ui.Must(fileutil.Remove(filepath.Join(dirname, ".terraform.lock.hcl")))
		ui.Must(fileutil.Remove(filepath.Join(dirname, "terraform.tf")))
		ui.Must(fileutil.Remove(filepath.Join(dirname, "versions.tf")))
		ui.Must(os.RemoveAll(filepath.Join(dirname, ".terraform")))
		for ; dirname != filepath.Join(terraform.RootModulesDirname, "network"); dirname = filepath.Dir(dirname) {
			if err := fileutil.Remove(dirname); err != nil {
				ui.Printf("couldn't remove %s (%s)", dirname, err)
				break
			}
		}

		vpcs0, err := awsec2.DescribeVPCs(ctx, networkCfg.Regional(region0), eq0.Environment, eq0.Quality)
		ui.Must(err)
		if len(vpcs0) != 1 { // TODO support sharing many VPCs when we introduce `substrate network create|delete|list`
			ui.Fatalf("expected 1 VPC but found %s", jsonutil.MustString(vpcs0))
		}
		vpc0 := vpcs0[0]
		vpcId0 := aws.ToString(vpc0.VpcId)
		//ui.Debug(vpc0)
		vpcs1, err := awsec2.DescribeVPCs(ctx, networkCfg.Regional(region1), eq1.Environment, eq1.Quality)
		ui.Must(err)
		if len(vpcs1) != 1 { // TODO support sharing many VPCs when we introduce `substrate network create|delete|list`
			ui.Fatalf("expected 1 VPC but found %s", jsonutil.MustString(vpcs1))
		}
		vpc1 := vpcs1[0]
		vpcId1 := aws.ToString(vpc1.VpcId)
		//ui.Debug(vpc1)
		conn, err := awsec2.EnsureVPCPeeringConnection(ctx, networkCfg, region0, vpcId0, region1, vpcId1)
		ui.Must(err)
		//ui.Debug(conn)
		ui.Spinf("routing traffic from %s to %s", vpcId0, vpcId1)
		public0, private0, err := awsec2.DescribeRouteTables(ctx, networkCfg.Regional(region0), vpcId0)
		ui.Must(err)
		//ui.Debug(public0, private0)
		ui.Must(awsec2.EnsureVPCPeeringRouteIPv4(
			ctx,
			networkCfg.Regional(region0),
			aws.ToString(public0.RouteTableId),
			cidr.MustIPv4(cidr.ParseIPv4(aws.ToString(vpc1.CidrBlockAssociationSet[0].CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
			ctx,
			networkCfg.Regional(region0),
			aws.ToString(public0.RouteTableId),
			cidr.MustIPv6(cidr.ParseIPv6(aws.ToString(vpc1.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		for _, rt := range private0 {
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv4(
				ctx,
				networkCfg.Regional(region0),
				aws.ToString(rt.RouteTableId),
				cidr.MustIPv4(cidr.ParseIPv4(aws.ToString(vpc1.CidrBlockAssociationSet[0].CidrBlock))),
				aws.ToString(conn.VpcPeeringConnectionId),
			))
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
				ctx,
				networkCfg.Regional(region0),
				aws.ToString(rt.RouteTableId),
				cidr.MustIPv6(cidr.ParseIPv6(aws.ToString(vpc1.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
				aws.ToString(conn.VpcPeeringConnectionId),
			))
		}
		ui.Stop("ok") // "routing traffic"
		ui.Spinf("routing traffic in reverse from %s to %s", vpcId1, vpcId0)
		public1, private1, err := awsec2.DescribeRouteTables(ctx, networkCfg.Regional(region1), vpcId1)
		ui.Must(err)
		//ui.Debug(public1, private1)
		ui.Must(awsec2.EnsureVPCPeeringRouteIPv4(
			ctx,
			networkCfg.Regional(region1),
			aws.ToString(public1.RouteTableId),
			cidr.MustIPv4(cidr.ParseIPv4(aws.ToString(vpc0.CidrBlockAssociationSet[0].CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
			ctx,
			networkCfg.Regional(region1),
			aws.ToString(public1.RouteTableId),
			cidr.MustIPv6(cidr.ParseIPv6(aws.ToString(vpc0.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		for _, rt := range private1 {
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv4(
				ctx,
				networkCfg.Regional(region1),
				aws.ToString(rt.RouteTableId),
				cidr.MustIPv4(cidr.ParseIPv4(aws.ToString(vpc0.CidrBlockAssociationSet[0].CidrBlock))),
				aws.ToString(conn.VpcPeeringConnectionId),
			))
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
				ctx,
				networkCfg.Regional(region1),
				aws.ToString(rt.RouteTableId),
				cidr.MustIPv6(cidr.ParseIPv6(aws.ToString(vpc0.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
				aws.ToString(conn.VpcPeeringConnectionId),
			))
		}
		ui.Stop("ok") // "routing traffic in reverse"

		ui.Stop("ok") // "peering"
	}
	// TODO remove the peering state files from S3 (on the region0 side)

}

func stateRmVPC(
	ctx context.Context,
	networkCfg *awscfg.Config,
	tags terraform.Tags,
	region string,
	natGateways bool,
	dirname string,
) {
	hasPrivateSubnets := tags.Environment != "admin"

	vpc := terraform.VPC{
		Label: terraform.Label(tags),
		Tags:  tags,
	}
	ui.Must(terraform.StateRm(dirname, vpc.Ref().Value()))

	// IPv4 and IPv6 Internet Gateways for the public subnets.
	igw := terraform.InternetGateway{
		Label: terraform.Label(tags),
		Tags:  tags,
		VpcId: terraform.U(vpc.Ref(), ".id"),
	}
	ui.Must(terraform.StateRm(dirname, igw.Ref().Value()))
	ui.Must(terraform.StateRm(dirname, terraform.Route{
		DestinationIPv4:   terraform.Q("0.0.0.0/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(tags, "public-internet-ipv4"),
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	}.Ref().Value()))
	ui.Must(terraform.StateRm(dirname, terraform.Route{
		DestinationIPv6:   terraform.Q("::/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(tags, "public-internet-ipv6"),
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	}.Ref().Value()))

	// IPv6 Egress-Only Internet Gateway for the private subnets.  (The IPv4
	// NAT Gateway comes later because it's per-subnet.  That is also where
	// this Egress-Only Internat Gateway is associated with the route table.)
	egw := terraform.EgressOnlyInternetGateway{
		Label: terraform.Label(tags),
		Tags:  tags,
		VpcId: terraform.U(vpc.Ref(), ".id"),
	}
	if hasPrivateSubnets {
		ui.Must(terraform.StateRm(dirname, egw.Ref().Value()))
	}

	// VPC Endpoint for S3, the one VPC Endpoint everyone's all but guaranteed to need.
	vpce := terraform.VPCEndpoint{
		Label: terraform.Label(tags),
		RouteTableIds: terraform.ValueSlice{
			terraform.U(vpc.Ref(), ".default_route_table_id"),
			// more will be appeneded before this function returns
		},
		ServiceName: terraform.Qf("com.amazonaws.%s.s3", region),
		Tags:        tags,
		VpcId:       terraform.U(vpc.Ref(), ".id"),
	}

	// Create a public and private subnet in each of (up to, and the newest)
	// three availability zones in the region.
	azs, err := availabilityzones.Select(ctx, networkCfg, region, availabilityzones.NumberPerNetwork)
	if err != nil {
		ui.Fatal(err)
	}
	for i, az := range azs {
		tags := terraform.Tags{
			AvailabilityZone: az,
			Environment:      tags.Environment,
			Name:             fmt.Sprintf("%s-%s", tags.Environment, tags.Quality),
			Quality:          tags.Quality,
			Region:           region,
			Special:          tags.Special,
		}

		// Public subnet, shared org-wide.
		bits := 2
		if hasPrivateSubnets {
			bits = 4
		}
		s := terraform.Subnet{
			AvailabilityZone:    terraform.Q(az),
			CidrBlock:           vpc.CidrsubnetIPv4(bits, i+1),
			IPv6CidrBlock:       vpc.CidrsubnetIPv6(8, i+1),
			Label:               terraform.Label(tags, "public"),
			MapPublicIPOnLaunch: true,
			Tags:                tags,
			VpcId:               terraform.U(vpc.Ref(), ".id"),
		}
		s.Tags.Connectivity = "public"
		s.Tags.Name = tags.Name + "-public-" + az
		ui.Must(terraform.StateRm(dirname, s.Ref().Value()))

		// Explicitly associate the public subnets with the main routing table.
		ui.Must(terraform.StateRm(dirname, terraform.RouteTableAssociation{
			Label:        s.Label,
			RouteTableId: terraform.U(vpc.Ref(), ".default_route_table_id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		}.Ref().Value()))

		if !hasPrivateSubnets {
			continue
		}

		// Save a reference to the public subnet in this availability zone
		// so we know where to put the NAT Gateway.
		natGatewaySubnetId := terraform.U(s.Ref(), ".id")

		// Private subnet, also shared org-wide.
		s = terraform.Subnet{
			AvailabilityZone: terraform.Q(az),
			CidrBlock:        vpc.CidrsubnetIPv4(2, i+1),
			IPv6CidrBlock:    vpc.CidrsubnetIPv6(8, i+0x81),
			Label:            terraform.Label(tags, "private"),
			Tags:             tags,
			VpcId:            terraform.U(vpc.Ref(), ".id"),
		}
		s.Tags.Connectivity = "private"
		s.Tags.Name = tags.Name + "-private-" + az
		ui.Must(terraform.StateRm(dirname, s.Ref().Value()))

		// Private subnets need their own routing tables to keep their NAT
		// Gateway traffic zonal.  The VPC Endpoint we created for S3 needs
		// to be made aware of this routing table, too.
		rt := terraform.RouteTable{
			Label: s.Label,
			Tags:  s.Tags,
			VpcId: terraform.U(vpc.Ref(), ".id"),
		}
		ui.Must(terraform.StateRm(dirname, rt.Ref().Value()))
		ui.Must(terraform.StateRm(dirname, terraform.RouteTableAssociation{
			Label:        s.Label,
			RouteTableId: terraform.U(rt.Ref(), ".id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		}.Ref().Value()))
		vpce.RouteTableIds = append(vpce.RouteTableIds, terraform.U(rt.Ref(), ".id"))

		// NAT Gateway for this private subnet.
		eip := terraform.EIP{
			Commented:          !natGateways,
			InternetGatewayRef: igw.Ref(),
			Label:              terraform.Label(tags),
			Tags:               tags,
		}
		eip.Tags.Name = tags.Name + "-nat-gateway-" + az
		ui.Must(terraform.StateRm(dirname, eip.Ref().Value()))
		ngw := terraform.NATGateway{
			Commented: !natGateways,
			Label:     terraform.Label(tags),
			SubnetId:  natGatewaySubnetId,
			Tags:      tags,
		}
		ngw.Tags.Name = tags.Name + "-" + az
		if natGateways {
			ui.Must(terraform.StateRm(dirname, ngw.Ref().Value()))
			ui.Must(terraform.StateRm(dirname, terraform.Route{
				Commented:       !natGateways,
				DestinationIPv4: terraform.Q("0.0.0.0/0"),
				Label:           terraform.Label(tags),
				NATGatewayId:    terraform.U(ngw.Ref(), ".id"),
				RouteTableId:    terraform.U(rt.Ref(), ".id"),
			}.Ref().Value()))
		}

		// Associate the VPC's Egress-Only Internet Gateway for IPv6 traffic.
		ui.Must(terraform.StateRm(dirname, terraform.Route{
			DestinationIPv6:             terraform.Q("::/0"),
			EgressOnlyInternetGatewayId: terraform.U(egw.Ref(), ".id"),
			Label:                       terraform.Label(s.Tags, "private-internet-ipv6"),
			RouteTableId:                terraform.U(rt.Ref(), ".id"),
		}.Ref().Value()))

	}

	// Now that all the route tables have been associated with the S3 VPC
	// Endpoint, add it to the file.
	ui.Must(terraform.StateRm(dirname, vpce.Ref().Value()))

}

func terraformVPC(
	ctx context.Context,
	mgmtCfg, networkCfg *awscfg.Config,
	environment, quality, region string,
	natGateways bool,
) {
	accountId := networkCfg.MustAccountId(ctx)
	dirname := filepath.Join(terraform.RootModulesDirname, accounts.Network, environment, quality, region)

	file := terraform.NewFile()
	org := terraform.Organization{
		Label: terraform.Q("current"),
	}
	file.Add(org)
	tags := terraform.Tags{
		Environment: environment,
		Name:        fmt.Sprintf("%s-%s", environment, quality),
		Quality:     quality,
		Region:      region,
	}
	// TODO data.aws_vpc.main? data.aws_vpc.shared? data.aws_vpc.vpc?
	// TODO data.aws_subnet
	// TODO data.aws_route_table
	ui.Must(file.Write(filepath.Join(dirname, "main.tf")))

	providersFile := terraform.NewFile()
	providersFile.Add(terraform.ProviderFor(
		region,
		roles.ARN(accountId, roles.NetworkAdministrator),
	))
	ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

	ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

	ui.Must(terraform.Fmt(dirname))

	ui.Must(terraform.Init(dirname))
	if *providersLock {
		ui.Must(terraform.ProvidersLock(dirname))
	}

	ui.Must(terraform.StateList(dirname)) // XXX
	stateRmVPC(ctx, networkCfg, tags, region, natGateways, dirname)
	ui.Must(terraform.StateList(dirname)) // XXX

	var err error
	if *noApply {
		err = terraform.Plan(dirname)
	} else {
		err = terraform.Apply(dirname, *autoApprove)
	}
	ui.Must(err)
}
