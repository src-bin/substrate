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
	ipv4 cidr.IPv4,
	natGateways bool,
) {

	ui.Spinf("finding or creating the %s %s VPC in %s", environment, quality, cfg.Region())
	vpc := ui.Must2(awsec2.EnsureVPC(ctx, cfg, environment, quality, ipv4, nil))
	//ui.Debug(vpc)
	vpcId := aws.ToString(vpc.VpcId)
	ipv6 := ui.Must2(cidr.ParseIPv6(aws.ToString(vpc.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)))
	// TODO remove all rules from the default security group
	ui.Stopf("%s %s %s", vpcId, ipv4, ipv6)

	azs := ui.Must2(availabilityzones.Select(ctx, cfg, cfg.Region(), availabilityzones.NumberPerNetwork))
	ui.Printf("using availability zones %s", strings.Join(azs, ", "))
	// TODO DescribeSubnets first and use those AZs because this selection will change if an AZ is added after Substrate is setup in a region

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

	igw := ui.Must2(awsec2.EnsureInternetGateway(ctx, cfg, vpcId, tagging.Map{
		tagging.Environment: environment,
		tagging.Name:        fmt.Sprintf("%s-%s", environment, quality),
		tagging.Quality:     quality,
	}))
	var eigw *awsec2.EgressOnlyInternetGateway
	if hasPrivateSubnets {
		eigw = ui.Must2(awsec2.EnsureEgressOnlyInternetGateway(ctx, cfg, vpcId, tagging.Map{
			tagging.Environment: environment,
			tagging.Name:        fmt.Sprintf("%s-%s", environment, quality),
			tagging.Quality:     quality,
		}))
	}

	// Three public and maybe three private subnets, too. One wasted subnet
	// at the beginning.
	publicRouteTable, privateRouteTables, err := awsec2.DescribeRouteTables(ctx, cfg, vpcId)
	ui.Must(err)
	//ui.Debug(publicRouteTable != nil, len(privateRouteTables))
	for i, az := range azs {
		ui.Spinf("finding or creating a public subnet in %s", az)

		publicSubnet := ui.Must2(awsec2.EnsureSubnet(
			ctx,
			cfg,
			vpcId,
			az,
			ui.Must2(ipv4.SubnetIPv4(bits, i+1)),
			ui.Must2(ipv6.SubnetIPv6(8, i+1)),
			tagging.Map{
				tagging.Connectivity: "public",
				tagging.Environment:  environment,
				tagging.Name:         fmt.Sprintf("%s-%s-public-%s", environment, quality, az),
				tagging.Quality:      quality,
			},
		))
		publicSubnetId := aws.ToString(publicSubnet.SubnetId)
		//ui.Debug(publicSubnet)

		ui.Must(awsec2.EnsureInternetGatewayRouteIPv4(
			ctx,
			cfg,
			aws.ToString(publicRouteTable.RouteTableId),
			ui.Must2(cidr.ParseIPv4("0.0.0.0/0")),
			aws.ToString(igw.InternetGatewayId),
		))
		ui.Must(awsec2.EnsureInternetGatewayRouteIPv6(
			ctx,
			cfg,
			aws.ToString(publicRouteTable.RouteTableId),
			ui.Must2(cidr.ParseIPv6("::/0")),
			aws.ToString(igw.InternetGatewayId),
		))

		ui.Stopf("%s %s %s", publicSubnetId, publicSubnet.CidrBlock, publicSubnet.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)

		if hasPrivateSubnets {
			ui.Spinf("finding or creating a private subnet in %s", az)
			subnetIPv4 := ui.Must2(ipv4.SubnetIPv4(2, i+1))
			subnetIPv6 := ui.Must2(ipv6.SubnetIPv6(8, i+0x81)) // to shift past the one wasted and three public subnets

			privateSubnet := ui.Must2(awsec2.EnsureSubnet(
				ctx,
				cfg,
				vpcId,
				az,
				subnetIPv4,
				subnetIPv6,
				tagging.Map{
					tagging.Connectivity: "private",
					tagging.Environment:  environment,
					tagging.Name:         fmt.Sprintf("%s-%s-private-%s", environment, quality, az),
					tagging.Quality:      quality,
				},
			))
			privateSubnetId := aws.ToString(privateSubnet.SubnetId)
			//ui.Debug(privateSubnet)

			if privateRouteTables[privateSubnetId] == nil {
				privateRouteTables[privateSubnetId] = ui.Must2(awsec2.CreateRouteTable(
					ctx,
					cfg,
					vpcId,
					privateSubnetId,
					tagging.Map{
						tagging.Connectivity: "private",
						tagging.Environment:  environment,
						tagging.Name:         fmt.Sprintf("%s-%s-private-%s", environment, quality, az),
						tagging.Quality:      quality,
					},
				))
			}

			if natGateways {
				ui.Spinf("finding or creating the NAT Gateway in %s (in %s for %s)", az, publicSubnetId, privateSubnetId)
				ngw := ui.Must2(awsec2.EnsureNATGateway(
					ctx,
					cfg,
					publicSubnetId,
					tagging.Map{
						tagging.Environment: environment,
						tagging.Name:        fmt.Sprintf("%s-%s", environment, quality),
						tagging.Quality:     quality,
					},
				))
				ui.Must(awsec2.EnsureNATGatewayRouteIPv4(
					ctx,
					cfg,
					aws.ToString(privateRouteTables[privateSubnetId].RouteTableId),
					ui.Must2(cidr.ParseIPv4("0.0.0.0/0")),
					aws.ToString(ngw.NatGatewayId),
				))
				ui.Stop(ngw.NatGatewayId)
			} else {
				ui.Spinf("deleting the NAT Gateway, if it exists, in %s", az)
				ui.Must(awsec2.DeleteRouteIPv4(
					ctx,
					cfg,
					aws.ToString(privateRouteTables[privateSubnetId].RouteTableId),
					ui.Must2(cidr.ParseIPv4("0.0.0.0/0")),
				))
				ui.Must(awsec2.DeleteNATGateway(ctx, cfg, publicSubnetId))
				ui.Stop("ok")
			}

			ui.Must(awsec2.EnsureEgressOnlyInternetGatewayRouteIPv6(
				ctx,
				cfg,
				aws.ToString(privateRouteTables[aws.ToString(privateSubnet.SubnetId)].RouteTableId),
				ui.Must2(cidr.ParseIPv6("::/0")),
				aws.ToString(eigw.EgressOnlyInternetGatewayId),
			))

			ui.Stopf("%s %s %s", privateSubnetId, privateSubnet.CidrBlock, privateSubnet.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock)
		}

	}
	publicRouteTable, privateRouteTables, err = awsec2.DescribeRouteTables(ctx, cfg, vpcId)
	ui.Must(err)
	//ui.Debug(publicRouteTable)
	//ui.Debug(privateRouteTables)
	routeTableIds := []string{aws.ToString(publicRouteTable.RouteTableId)}
	for _, rt := range privateRouteTables {
		routeTableIds = append(routeTableIds, aws.ToString(rt.RouteTableId))
	}

	ui.Spin("finding or creating gateway VPC Endpoints for DynamoDB and S3 (these are free)")
	for _, serviceName := range []string{
		fmt.Sprintf("com.amazonaws.%s.dynamodb", cfg.Region()),
		fmt.Sprintf("com.amazonaws.%s.s3", cfg.Region()),
	} {
		ui.Must(awsec2.EnsureGatewayVPCEndpoint(
			ctx,
			cfg,
			vpcId,
			routeTableIds,
			serviceName,
			tagging.Map{
				tagging.Environment: environment,
				tagging.Name:        fmt.Sprintf("%s-%s", environment, quality),
				tagging.Quality:     quality,
			},
		))
	}
	ui.Stop("ok")

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
		`do you want to provision NAT Gateways for IPv4 traffic from your private subnets to the Internet? (yes/no; answering "yes" costs about $108 per month per region per environment/quality pair)`,
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

	// TODO delete the default VPC in every region to free up some quota

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
			ui.Must2(cidr.ParseIPv4(aws.ToString(vpc1.CidrBlockAssociationSet[0].CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
			ctx,
			networkCfg.Regional(region0),
			aws.ToString(public0.RouteTableId),
			ui.Must2(cidr.ParseIPv6(aws.ToString(vpc1.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		for _, rt := range private0 {
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv4(
				ctx,
				networkCfg.Regional(region0),
				aws.ToString(rt.RouteTableId),
				ui.Must2(cidr.ParseIPv4(aws.ToString(vpc1.CidrBlockAssociationSet[0].CidrBlock))),
				aws.ToString(conn.VpcPeeringConnectionId),
			))
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
				ctx,
				networkCfg.Regional(region0),
				aws.ToString(rt.RouteTableId),
				ui.Must2(cidr.ParseIPv6(aws.ToString(vpc1.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
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
			ui.Must2(cidr.ParseIPv4(aws.ToString(vpc0.CidrBlockAssociationSet[0].CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
			ctx,
			networkCfg.Regional(region1),
			aws.ToString(public1.RouteTableId),
			ui.Must2(cidr.ParseIPv6(aws.ToString(vpc0.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
			aws.ToString(conn.VpcPeeringConnectionId),
		))
		for _, rt := range private1 {
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv4(
				ctx,
				networkCfg.Regional(region1),
				aws.ToString(rt.RouteTableId),
				ui.Must2(cidr.ParseIPv4(aws.ToString(vpc0.CidrBlockAssociationSet[0].CidrBlock))),
				aws.ToString(conn.VpcPeeringConnectionId),
			))
			ui.Must(awsec2.EnsureVPCPeeringRouteIPv6(
				ctx,
				networkCfg.Regional(region1),
				aws.ToString(rt.RouteTableId),
				ui.Must2(cidr.ParseIPv6(aws.ToString(vpc0.Ipv6CidrBlockAssociationSet[0].Ipv6CidrBlock))),
				aws.ToString(conn.VpcPeeringConnectionId),
			))
		}
		ui.Stop("ok") // "routing traffic in reverse"

		ui.Stop("ok") // "peering"
	}
	// TODO remove the peering state files from S3 (on the region0 side)

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
	/*
		tags := terraform.Tags{
			Environment: environment,
			Name:        fmt.Sprintf("%s-%s", environment, quality),
			Quality:     quality,
			Region:      region,
		}
	*/
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

	if *runTerraform {
		ui.Must(terraform.Init(dirname))
		if *providersLock {
			ui.Must(terraform.ProvidersLock(dirname))
		}
		if *noApply {
			ui.Must(terraform.Plan(dirname))
		} else {
			ui.Must(terraform.Apply(dirname, *autoApprove))
		}
	}
}
