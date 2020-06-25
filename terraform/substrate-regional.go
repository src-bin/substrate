package terraform

// managed by go generate; do not edit by hand

func substrateRegionalTemplate() map[string]string {
	return map[string]string{
		"providers.tf": `provider "aws" { alias = "global" }

provider "aws" { alias = "network" }
`,
		"outputs.tf":   `output "tags" {
  value = module.global.tags
}

output "private_subnet_ids" {
  value = data.aws_subnet_ids.private.ids
}

output "public_subnet_ids" {
  value = data.aws_subnet_ids.public.ids
}

output "vpc_id" {
  value = data.aws_vpc.network.id
}
`,
		"vpc.tf":       `data "aws_subnet_ids" "private" {
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
`,
		"global.tf":    `module "global" {
  providers = { aws.global = aws.global }
  source    = "../global"
}
`,
	}
}
