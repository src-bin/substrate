package main

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/src-bin/substrate/accounts"
	"github.com/src-bin/substrate/availabilityzones"
	"github.com/src-bin/substrate/awsservicequotas"
	"github.com/src-bin/substrate/awssessions"
	"github.com/src-bin/substrate/awssts"
	"github.com/src-bin/substrate/awsutil"
	"github.com/src-bin/substrate/networks"
	"github.com/src-bin/substrate/regions"
	"github.com/src-bin/substrate/roles"
	"github.com/src-bin/substrate/terraform"
	"github.com/src-bin/substrate/ui"
	"github.com/src-bin/substrate/veqp"
)

const (
	EnvironmentsFilename = "substrate.Environments"
	QualitiesFilename    = "substrate.Qualities"
	TerraformDirname     = "network-account"
)

func main() {

	sess, err := awssessions.Special(
		accounts.Network,
		roles.NetworkAdministrator,
		awssessions.Config{},
	)
	if err != nil {
		ui.Print("unable to assume the NetworkAdministrator role, which means this is probably your first time bootstrapping your networks; please provide an access key from your master AWS account")
		accessKeyId, secretAccessKey := awsutil.ReadAccessKeyFromStdin()
		ui.Printf("using access key %s", accessKeyId)
		sess, err = awssessions.Special(
			accounts.Network,
			roles.NetworkAdministrator,
			awssessions.Config{
				AccessKeyId:     accessKeyId,
				SecretAccessKey: secretAccessKey,
			},
		)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Gather the definitive list of Environments and Qualities first.
	environments, err := ui.EditFile(
		EnvironmentsFilename,
		"the following Environments are currently valid in your Substrate-managed infrastructure:",
		`list all your Environments, one per line, in order of progression from e.g. development through e.g. production; your list MUST include "admin"`,
	)
	if err != nil {
		log.Fatal(err)
	}
	found := false
	for _, environment := range environments {
		found = found || environment == "admin"
	}
	if !found {
		ui.Print(`you must include "admin" in your list of Environments`)
		return
	}
	ui.Printf("using Environments %s", strings.Join(environments, ", "))
	qualities, err := ui.EditFile(
		QualitiesFilename,
		"the following Qualities are currently valid in your Substrate-managed infrastructure:",
		`list all your Qualities, one per line, in order from least to greatest quality (Substrate recommends "alpha", "beta", and "gamma")`,
	)
	if err != nil {
		log.Fatal(err)
	}
	if len(qualities) < 2 {
		ui.Print(`you must define at least two Qualities (and Substrate recommends "alpha", "beta", and "gamma")`)
		return
	}
	ui.Printf("using Qualities %s", strings.Join(qualities, ", "))

	// Combine all Environments and Qualities.  If a given combination doesn't
	// appear in substrate.ValidEnvironmentQualityPairs.json then offer its
	// inclusion before validating the final document.
	veqpDoc, err := veqp.ReadDocument()
	if err != nil {
		log.Fatal(err)
	}
	for _, environment := range environments {
		for _, quality := range qualities {
			if !veqpDoc.Valid(environment, quality) {
				ok, err := ui.Confirmf(`do you want to allow %s-Quality infrastructure in your %s Environment?`, quality, environment)
				if err != nil {
					log.Fatal(err)
				}
				if ok {
					veqpDoc.Ensure(environment, quality)
				}
			}
		}
	}
	if err := veqpDoc.Validate(environments, qualities); err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", veqpDoc)

	// Select or confirm which regions to use.
	if _, err := regions.Select(); err != nil {
		log.Fatal(err)
	}

	// Configure the allocator for admin networks to use 192.168.0.0/16 and
	// 21-bit subnet masks which yields 2,048 IP addresses per VPC and 32
	// possible VPCs while keeping a tidy source IP address range for granting
	// SSH and other administrative access safely and easily.
	adminNetDoc, err := networks.ReadDocument(networks.AdminFilename, networks.RFC1918_192_168_0_0_16, 21)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", adminNetDoc)

	// Configure the allocator for normal (Environment, Quality) networks to use
	// 10.0.0.0/8 and 18-bit subnet masks which yields 16,384 IP addresses per
	// VPC and 1,024 possible VPCs.
	netDoc, err := networks.ReadDocument(networks.Filename, networks.RFC1918_10_0_0_0_8, 18)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("%+v", netDoc)

	// Provide every Terraform module with a reference to the organization.
	orgBlocks := terraform.NewBlocks()
	org := terraform.Organization{
		Label:    terraform.Q("current"),
		Provider: terraform.ProviderAliasFor(regions.Selected()[0]),
	}
	orgBlocks.Push(org)

	// Write (or rewrite) some Terraform providers to make everything usable.
	callerIdentity := awssts.MustGetCallerIdentity(sts.New(sess))
	providers := terraform.Provider{
		AccountId:   aws.StringValue(callerIdentity.Account),
		RoleName:    roles.NetworkAdministrator,
		SessionName: "Terraform",
	}.AllRegions()

	// Write (or rewrite) Terraform resources that create the various
	// (Environment, Quality) networks.  Networks in the admin Environment will
	// be created in the 192.168.0.0/16 CIDR block managed by adminNetDoc.
	ui.Printf("bootstrapping networks for every Environment and Quality in %d regions", len(regions.Selected()))
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		blocks := terraform.NewBlocks()

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
				Environment: eq.Environment,
				Quality:     eq.Quality,
				Region:      region,
			})
			if err != nil {
				log.Fatal(err)
			}
			//log.Printf("%+v", net)

			tags := terraform.Tags{
				Environment: eq.Environment,
				Name:        fmt.Sprintf("%s-%s", eq.Environment, eq.Quality),
				Quality:     eq.Quality,
				Region:      region,
			}
			vpc := terraform.VPC{
				CidrBlock: terraform.Q(n.IPv4.String()),
				Label:     terraform.Label(tags),
				Provider:  terraform.ProviderAliasFor(region),
				Tags:      tags,
			}
			blocks.Push(vpc)

			vpcAccoutrements(sess, region, org, vpc, blocks)

			ui.Stop(n.IPv4)
		}

		if err := orgBlocks.Write(path.Join(TerraformDirname, eq.Environment, eq.Quality, "organization.tf")); err != nil {
			log.Fatal(err)
		}
		if err := providers.Write(path.Join(TerraformDirname, eq.Environment, eq.Quality, "providers.tf")); err != nil {
			log.Fatal(err)
		}
		if err := blocks.Write(path.Join(TerraformDirname, eq.Environment, eq.Quality, "vpc.tf")); err != nil {
			log.Fatal(err)
		}

	}

	// Write to substrate.admin-networks.json and substrate.networks.json once
	// more so that, even if no changes were made, formatting changes and
	// SubstrateVersion are changed.
	if err := adminNetDoc.Write(); err != nil {
		log.Fatal(err)
	}
	if err := netDoc.Write(); err != nil {
		log.Fatal(err)
	}

	// Format all the Terraform code you can possibly find.
	if err := terraform.Fmt(); err != nil {
		log.Fatal(err)
	}

	// Ensure the VPCs-per-region service quota and a few others that  isn't going to get in the way.
	ui.Print("raising the VPC, Internet, Egress-Only Internet, and NAT Gateway, and EIP service quotas in all your regions (this could take days, unfortunately; this program is safe to re-run)")
	desiredValue := float64(len(netDoc.FindAll(&networks.Network{Region: regions.Selected()[0]})) + // networks for existing (Environment, Quality) pairs
		len(qualities) + // plus room to add another Environment
		1 + // plus the ops network
		1) // plus the default VPC
	for _, quota := range [][2]string{
		{"L-F678F1CE", "vpc"}, // VPCs per region
		{"L-45FE3B85", "vpc"}, // Egress-Only Internet Gateways per region
		{"L-A4707A72", "vpc"}, // Internet Gateways per region
		{"L-FE5A380F", "vpc"}, // NAT Gateways per availability zone
		// XXX not waiting on this now, just seeing what breaks {"L-0263D0A3", "ec2"}, // EIPs per VPC
	} {
		if err := awsservicequotas.EnsureServiceQuotaInAllRegions(
			sess,
			quota[0],
			quota[1],
			desiredValue,
		); err != nil {
			log.Fatal(err)
		}
	}

	// Generate a Makefile in each root Terraform module then apply the generated
	// Terraform code.  Start with the ops networks, then move on to the
	// Environments, all Quality-by-Quality with a pause in between.
	// TODO confirmation between steps
	for _, eq := range veqpDoc.ValidEnvironmentQualityPairs {
		dirname := path.Join(TerraformDirname, eq.Environment, eq.Quality)
		if err := terraform.Makefile(dirname); err != nil {
			log.Fatal(err)
		}
		if err := terraform.Init(dirname); err != nil {
			log.Fatal(err)
		}
		if err := terraform.Apply(dirname); err != nil {
			log.Fatal(err)
		}
	}

}

func vpcAccoutrements(
	sess *session.Session,
	region string,
	org terraform.Organization,
	vpc terraform.VPC,
	blocks *terraform.Blocks,
) {
	hasPrivateSubnets := vpc.Tags.Environment != "admin"

	// Accept the default Network ACL until we need to do otherwise.

	// TODO manage the default security group to ensure it has no rules.

	// Accept the default DHCP option set until we need to do otherwise.

	// A resource share for the subnets to reference, shared org-wide.
	rs := terraform.ResourceShare{
		Label:    terraform.Label(vpc.Tags),
		Provider: vpc.Provider,
		Tags:     vpc.Tags,
	}
	blocks.Push(rs)
	blocks.Push(terraform.PrincipalAssociation{
		Label:            terraform.Label(vpc.Tags),
		Principal:        terraform.U(org.Ref(), ".arn"),
		Provider:         vpc.Provider,
		ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
	})

	// IPv4 and IPv6 Internet Gateways for the public subnets.
	igw := terraform.InternetGateway{
		Label:    terraform.Label(vpc.Tags),
		Provider: vpc.Provider,
		Tags:     vpc.Tags,
		VpcId:    terraform.U(vpc.Ref(), ".id"),
	}
	blocks.Push(igw)
	blocks.Push(terraform.Route{
		DestinationIPv4:   terraform.Q("0.0.0.0/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(vpc.Tags, "public-internet-ipv4"),
		Provider:          vpc.Provider,
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	})
	blocks.Push(terraform.Route{
		DestinationIPv6:   terraform.Q("::/0"),
		InternetGatewayId: terraform.U(igw.Ref(), ".id"),
		Label:             terraform.Label(vpc.Tags, "public-internet-ipv6"),
		Provider:          vpc.Provider,
		RouteTableId:      terraform.U(vpc.Ref(), ".default_route_table_id"),
	})

	// IPv6 Egress-Only Internet Gateway for the private subnets.  (The IPv4
	// NAT Gateway comes later because it's per-subnet.  That is also where
	// this Egress-Only Internat Gateway is associated with the route table.)
	egw := terraform.EgressOnlyInternetGateway{
		Label:    terraform.Label(vpc.Tags),
		Provider: vpc.Provider,
		Tags:     vpc.Tags,
		VpcId:    terraform.U(vpc.Ref(), ".id"),
	}
	if hasPrivateSubnets {
		blocks.Push(egw)
	}

	// Create a public and private subnet in each of (up to, and the newest)
	// three availability zones in the region.
	azs, err := availabilityzones.Select(sess, region, 3)
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
			Provider:            vpc.Provider,
			Tags:                tags,
			VpcId:               terraform.U(vpc.Ref(), ".id"),
		}
		s.Tags.Name = vpc.Tags.Name + "-public-" + az
		blocks.Push(s)
		blocks.Push(terraform.ResourceAssociation{
			Label:            s.Label,
			Provider:         vpc.Provider,
			ResourceArn:      terraform.U(s.Ref(), ".arn"),
			ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
		})

		// Explicitly associate the public subnets with the main routing table.
		blocks.Push(terraform.RouteTableAssociation{
			Label:        s.Label,
			Provider:     vpc.Provider,
			RouteTableId: terraform.U(vpc.Ref(), ".default_route_table_id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		})

		if !hasPrivateSubnets {
			continue
		}

		// Private subnet, also shared org-wide.
		s = terraform.Subnet{
			AvailabilityZone: terraform.Q(az),
			CidrBlock:        vpc.CidrsubnetIPv4(2, i+1),
			IPv6CidrBlock:    vpc.CidrsubnetIPv6(8, i+0x81),
			Label:            terraform.Label(tags, "private"),
			Provider:         vpc.Provider,
			Tags:             tags,
			VpcId:            terraform.U(vpc.Ref(), ".id"),
		}
		s.Tags.Name = vpc.Tags.Name + "-private-" + az
		blocks.Push(s)
		blocks.Push(terraform.ResourceAssociation{
			Label:            s.Label,
			Provider:         vpc.Provider,
			ResourceArn:      terraform.U(s.Ref(), ".arn"),
			ResourceShareArn: terraform.U(rs.Ref(), ".arn"),
		})

		// Private subnets need their own routing tables to keep their NAT
		// Gateway traffic zonal.
		rt := terraform.RouteTable{
			Label:    s.Label,
			Provider: vpc.Provider,
			Tags:     s.Tags,
			VpcId:    terraform.U(vpc.Ref(), ".id"),
		}
		blocks.Push(rt)
		blocks.Push(terraform.RouteTableAssociation{
			Label:        s.Label,
			Provider:     vpc.Provider,
			RouteTableId: terraform.U(rt.Ref(), ".id"),
			SubnetId:     terraform.U(s.Ref(), ".id"),
		})

		// NAT Gateway for this private subnet.
		eip := terraform.EIP{
			InternetGatewayRef: igw.Ref(),
			Label:              terraform.Label(tags),
			Provider:           vpc.Provider,
			Tags:               tags,
		}
		eip.Tags.Name = vpc.Tags.Name + "-nat-gateway-" + az
		blocks.Push(eip)
		ngw := terraform.NATGateway{
			Label:    terraform.Label(tags),
			Provider: vpc.Provider,
			SubnetId: terraform.U(s.Ref(), ".id"),
			Tags:     tags,
		}
		ngw.Tags.Name = vpc.Tags.Name + "-" + az
		blocks.Push(ngw)
		blocks.Push(terraform.Route{
			DestinationIPv4: terraform.Q("0.0.0.0/0"),
			Label:           terraform.Label(tags),
			NATGatewayId:    terraform.U(ngw.Ref(), ".id"),
			Provider:        vpc.Provider,
			RouteTableId:    terraform.U(rt.Ref(), ".id"),
		})

		// Associate the VPC's Egress-Only Internet Gateway for IPv6 traffic.
		blocks.Push(terraform.Route{
			DestinationIPv6:             terraform.Q("::/0"),
			EgressOnlyInternetGatewayId: terraform.U(egw.Ref(), ".id"),
			Label:                       terraform.Label(s.Tags, "private-internet-ipv6"),
			Provider:                    vpc.Provider,
			RouteTableId:                terraform.U(rt.Ref(), ".id"),
		})

	}

}
