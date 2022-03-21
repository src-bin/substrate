package bootstrapnetworkaccount

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/src-bin/substrate/availabilityzones"
	"github.com/src-bin/substrate/terraform"
)

func vpcAccoutrements(
	sess *session.Session,
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
	file.Push(igw)
	file.Push(terraform.Route{
		DestinationIPv4:   terraform.Q("0.0.0.0/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(vpc.Tags, "public-internet-ipv4"),
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	})
	file.Push(terraform.Route{
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
		file.Push(egw)
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
	azs, err := availabilityzones.Select(sess, region, availabilityzones.NumberPerNetwork)
	if err != nil {
		log.Fatal(err)
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
		file.Push(s)

		// Explicitly associate the public subnets with the main routing table.
		file.Push(terraform.RouteTableAssociation{
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
		file.Push(s)

		// Private subnets need their own routing tables to keep their NAT
		// Gateway traffic zonal.  The VPC Endpoint we created for S3 needs
		// to be made aware of this routing table, too.
		rt := terraform.RouteTable{
			Label: s.Label,
			Tags:  s.Tags,
			VpcId: terraform.U(vpc.Ref(), ".id"),
		}
		file.Push(rt)
		file.Push(terraform.RouteTableAssociation{
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
		file.Push(eip)
		ngw := terraform.NATGateway{
			Commented: !natGateways,
			Label:     terraform.Label(tags),
			SubnetId:  natGatewaySubnetId,
			Tags:      tags,
		}
		ngw.Tags.Name = vpc.Tags.Name + "-" + az
		file.Push(ngw)
		file.Push(terraform.Route{
			Commented:       !natGateways,
			DestinationIPv4: terraform.Q("0.0.0.0/0"),
			Label:           terraform.Label(tags),
			NATGatewayId:    terraform.U(ngw.Ref(), ".id"),
			RouteTableId:    terraform.U(rt.Ref(), ".id"),
		})

		// Associate the VPC's Egress-Only Internet Gateway for IPv6 traffic.
		file.Push(terraform.Route{
			DestinationIPv6:             terraform.Q("::/0"),
			EgressOnlyInternetGatewayId: terraform.U(egw.Ref(), ".id"),
			Label:                       terraform.Label(s.Tags, "private-internet-ipv6"),
			RouteTableId:                terraform.U(rt.Ref(), ".id"),
		})

	}

	// Now that all the route tables have been associated with the S3 VPC
	// Endpoint, add it to the file.
	file.Push(vpce)

}
