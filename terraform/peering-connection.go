package terraform

// managed by go generate; do not edit by hand

func peeringConnectionTemplate() map[string]string {
	return map[string]string{
		"providers.tf": `provider "aws" { alias = "accepter" }

provider "aws" { alias = "requester" }
`,
		"variables.tf": `variable "accepter_quality" {
  type = string
}

variable "environment" {
  type = string
}

variable "requester_quality" {
  type = string
}
`,
		"main.tf":      `data "aws_region" "accepter" {}

/*
data "aws_subnet_ids" "private" {
  count    = module.global.tags.environment == "admin" ? 0 : 1
  provider = aws.network
  tags = {
    Connectivity = "private"
    Environment  = module.global.tags.environment
    Quality      = module.global.tags.quality
  }
  vpc_id = data.aws_vpc.network.id
}

data "aws_subnet_ids" "public" {
  provider = aws.network
  tags = {
    Connectivity = "public"
    Environment  = module.global.tags.environment
    Quality      = module.global.tags.quality
  }
  vpc_id = data.aws_vpc.network.id
}
*/

data "aws_vpc" "accepter" {
  provider = aws.accepter
  tags = {
    Environment = var.environment
    Quality     = var.accepter_quality
  }
}

data "aws_vpc" "requester" {
  provider = aws.requester
  tags = {
    Environment = var.environment
    Quality     = var.requester_quality
  }
}

resource "aws_vpc_peering_connection" "requester" {
  auto_accept = false
  peer_region = data.aws_region.accepter.name
  peer_vpc_id = data.aws_vpc.accepter.id
  provider    = aws.requester
  tags = {
    Manager = "Terraform"
  }
  vpc_id = data.aws_vpc.requester.id
}

resource "aws_vpc_peering_connection_accepter" "accepter" {
  auto_accept = true
  provider    = aws.accepter
  tags = {
    Manager = "Terraform"
  }
  vpc_peering_connection_id = aws_vpc_peering_connection.requester.id
}

resource "aws_vpc_peering_connection_options" "accepter" {
  provider = aws.accepter
  requester {
    allow_remote_vpc_dns_resolution = true
  }
  vpc_peering_connection_id = aws_vpc_peering_connection_accepter.accepter.id
}

resource "aws_vpc_peering_connection_options" "requester" {
  accepter {
    allow_remote_vpc_dns_resolution = true
  }
  provider                  = aws.requester
  vpc_peering_connection_id = aws_vpc_peering_connection_accepter.accepter.id
}
`,
	}
}
