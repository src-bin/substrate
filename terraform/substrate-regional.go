package terraform

// managed by go generate; do not edit by hand

func substrateRegionalTemplate() map[string]string {
	return map[string]string{
		"main.tf":    `data "aws_subnet_ids" "private" {
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

data "aws_vpc" "network" {
  provider = aws.network
  tags = {
    Environment = module.global.tags.environment
    Quality     = module.global.tags.quality
  }
}

module "global" {
  source = "../global"
}
`,
		"outputs.tf": `output "tags" {
  value = module.global.tags
}

output "private_subnet_ids" {
  value = module.global.tags.environment == "admin" ? [] : data.aws_subnet_ids.private[0].ids
}

output "public_subnet_ids" {
  value = data.aws_subnet_ids.public.ids
}

output "vpc_id" {
  value = data.aws_vpc.network.id
}
`,
	}
}
