package setup

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/availabilityzones"
	"github.com/src-bin/substrate/awscfg"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

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

	// Write (or rewrite) Terraform resources that create the various
	// (environment, quality) networks.  Networks in the admin environment will
	// be created in the 192.168.0.0/16 CIDR block managed by adminNetDoc.
	ui.Printf("configuring networks for every environment and quality in %d regions", len(regions.Selected()))
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
			//log.Printf("%+v", net)
			ui.Stop(n.IPv4)

			dirname := filepath.Join(terraform.RootModulesDirname, accounts.Network, eq.Environment, eq.Quality, region)

			file := terraform.NewFile()
			org := terraform.Organization{
				Label: terraform.Q("current"),
			}
			file.Add(org)
			tags := terraform.Tags{
				Environment: eq.Environment,
				Name:        fmt.Sprintf("%s-%s", eq.Environment, eq.Quality),
				Quality:     eq.Quality,
				Region:      region,
			}
			vpc := terraform.VPC{
				CidrBlock: terraform.Q(n.IPv4.String()),
				Label:     terraform.Label(tags),
				Tags:      tags,
			}
			file.Add(vpc)
			vpcAccoutrements(ctx, cfg, natGateways, region, org, vpc, file)
			ui.Must(file.Write(filepath.Join(dirname, "main.tf")))

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
	if !*autoApprove && !*noApply {
		ui.Print("this tool can affect multiple environments and qualities in rapid succession")
		ui.Print("for safety's sake, it will pause for confirmation before proceeding with each enviornment and quality")
	}
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		for _, region := range regions.Selected() {
			dirname := filepath.Join(terraform.RootModulesDirname, accounts.Network, eq.Environment, eq.Quality, region)

			providersFile := terraform.NewFile()

			// The default provider for building out networks in this root module.
			providersFile.Add(terraform.ProviderFor(
				region,
				roles.ARN(accountId, roles.NetworkAdministrator),
			))

			// A provider for the substrate module to use, if for some reason it's
			// desired in this context.
			providersFile.Add(terraform.NetworkProviderFor(
				region,
				roles.ARN(accountId, roles.Auditor),
			))
			ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

			ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region))

			ui.Must(terraform.Fmt(dirname))

			ui.Must(terraform.Init(dirname))

			if *noApply {
				err = terraform.Plan(dirname)
			} else {
				err = terraform.Apply(dirname, *autoApprove)
			}
			ui.Must(err)
		}
	}

	// Now that all the networks exist, establish a fully-connected mesh of
	// peering connections within each environment's qualities and regions.
	peeringConnectionModule := terraform.PeeringConnectionModule()
	ui.Must(peeringConnectionModule.Write(filepath.Join(terraform.ModulesDirname, "peering-connection")))
	peeringConnections, err := networks.EnumeratePeeringConnections()
	ui.Must(err)
	for _, pc := range peeringConnections.Slice() {
		eq0, eq1, region0, region1 := pc.Ends()

		ui.Printf(
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

		file := terraform.NewFile()
		file.Add(terraform.Module{
			Arguments: map[string]terraform.Value{
				"accepter_environment":  terraform.Q(eq0.Environment),
				"accepter_quality":      terraform.Q(eq0.Quality),
				"requester_environment": terraform.Q(eq1.Environment),
				"requester_quality":     terraform.Q(eq1.Quality),
			},
			Label: terraform.Q("peering-connection"),
			Providers: map[terraform.ProviderAlias]terraform.ProviderAlias{
				terraform.ProviderAliasFor("accepter"):  terraform.ProviderAliasFor("accepter"),
				terraform.ProviderAliasFor("requester"): terraform.ProviderAliasFor("requester"),
			},
			Source: terraform.Q("../../../../../../../../../modules/peering-connection"),
		})
		ui.Must(file.Write(filepath.Join(dirname, "main.tf")))

		providersFile := terraform.NewFile()
		accepterProvider := terraform.ProviderFor(
			region1,
			roles.ARN(accountId, roles.NetworkAdministrator),
		)
		accepterProvider.Alias = "accepter"
		providersFile.Add(accepterProvider)
		requesterProvider := terraform.ProviderFor(
			region0,
			roles.ARN(accountId, roles.NetworkAdministrator),
		)
		requesterProvider.Alias = "requester"
		providersFile.Add(requesterProvider)
		ui.Must(providersFile.Write(filepath.Join(dirname, "providers.tf")))

		// The choice of region0 here is arbitrary.  Only one side
		// can store the Terraform state and region0 wins.
		ui.Must(terraform.Root(ctx, mgmtCfg, dirname, region0))

		ui.Must(terraform.Fmt(dirname))

		ui.Must(terraform.Init(dirname))

		if *noApply {
			err = terraform.Plan(dirname)
		} else {
			err = terraform.Apply(dirname, true) // always auto-approve peering since it's low-stakes and high-annoyance
		}
		ui.Must(err)
	}
}

func vpcAccoutrements(
	ctx context.Context,
	cfg *awscfg.Config,
	natGateways bool,
	region string,
	org terraform.Organization,
	vpc terraform.VPC,
	file *terraform.File,
) {
	hasPrivateSubnets := vpc.Tags.Environment != "admin"

	// Accept the default Network ACL until we need to do otherwise.

	// TODO manage the default security group to ensure it has no rules.

	// Accept the default DHCP option set until we need to do otherwise.

	// IPv4 and IPv6 Internet Gateways for the public subnets.
	igw := terraform.InternetGateway{
		Label: terraform.Label(vpc.Tags),
		Tags:  vpc.Tags,
		VpcId: terraform.U(vpc.Ref(), ".id"),
	}
	file.Add(igw)
	file.Add(terraform.Route{
		DestinationIPv4:   terraform.Q("0.0.0.0/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(vpc.Tags, "public-internet-ipv4"),
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	})
	file.Add(terraform.Route{
		DestinationIPv6:   terraform.Q("::/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(vpc.Tags, "public-internet-ipv6"),
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	})

	// IPv6 Egress-Only Internet Gateway for the private subnets.  (The IPv4
	// NAT Gateway comes later because it's per-subnet.  That is also where
	// this Egress-Only Internat Gateway is associated with the route table.)
	egw := terraform.EgressOnlyInternetGateway{
		Label: terraform.Label(vpc.Tags),
		Tags:  vpc.Tags,
		VpcId: terraform.U(vpc.Ref(), ".id"),
	}
	if hasPrivateSubnets {
		file.Add(egw)
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

	// Create a public and private subnet in each of (up to, and the newest)
	// three availability zones in the region.
	azs, err := availabilityzones.Select(ctx, cfg, region, availabilityzones.NumberPerNetwork)
	if err != nil {
		ui.Fatal(err)
	}
	for i, az := range azs {
		tags := terraform.Tags{
			AvailabilityZone: az,
			Environment:      vpc.Tags.Environment,
			Quality:          vpc.Tags.Quality,
			Region:           region,
			Special:          vpc.Tags.Special,
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
		s.Tags.Name = vpc.Tags.Name + "-public-" + az
		file.Add(s)

		// Explicitly associate the public subnets with the main routing table.
		file.Add(terraform.RouteTableAssociation{
			Label:        s.Label,
			RouteTableId: terraform.U(vpc.Ref(), ".default_route_table_id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		})

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
		s.Tags.Name = vpc.Tags.Name + "-private-" + az
		file.Add(s)

		// Private subnets need their own routing tables to keep their NAT
		// Gateway traffic zonal.  The VPC Endpoint we created for S3 needs
		// to be made aware of this routing table, too.
		rt := terraform.RouteTable{
			Label: s.Label,
			Tags:  s.Tags,
			VpcId: terraform.U(vpc.Ref(), ".id"),
		}
		file.Add(rt)
		file.Add(terraform.RouteTableAssociation{
			Label:        s.Label,
			RouteTableId: terraform.U(rt.Ref(), ".id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		})
		vpce.RouteTableIds = append(vpce.RouteTableIds, terraform.U(rt.Ref(), ".id"))

		// NAT Gateway for this private subnet.
		eip := terraform.EIP{
			Commented:          !natGateways,
			InternetGatewayRef: igw.Ref(),
			Label:              terraform.Label(tags),
			Tags:               tags,
		}
		eip.Tags.Name = vpc.Tags.Name + "-nat-gateway-" + az
		file.Add(eip)
		ngw := terraform.NATGateway{
			Commented: !natGateways,
			Label:     terraform.Label(tags),
			SubnetId:  natGatewaySubnetId,
			Tags:      tags,
		}
		ngw.Tags.Name = vpc.Tags.Name + "-" + az
		file.Add(ngw)
		file.Add(terraform.Route{
			Commented:       !natGateways,
			DestinationIPv4: terraform.Q("0.0.0.0/0"),
			Label:           terraform.Label(tags),
			NATGatewayId:    terraform.U(ngw.Ref(), ".id"),
			RouteTableId:    terraform.U(rt.Ref(), ".id"),
		})

		// Associate the VPC's Egress-Only Internet Gateway for IPv6 traffic.
		file.Add(terraform.Route{
			DestinationIPv6:             terraform.Q("::/0"),
			EgressOnlyInternetGatewayId: terraform.U(egw.Ref(), ".id"),
			Label:                       terraform.Label(s.Tags, "private-internet-ipv6"),
			RouteTableId:                terraform.U(rt.Ref(), ".id"),
		})

	}

	// Now that all the route tables have been associated with the S3 VPC
	// Endpoint, add it to the file.
	file.Add(vpce)

}
