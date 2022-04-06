data "aws_subnets" "private" {
  count = module.global.tags.environment == "admin" ? 0 : 1
  filter {
    name   = "tag:Connectivity"
    values = ["private"]
  }
  filter {
    name   = "tag:Environment"
    values = [module.global.tags.environment]
  }
  filter {
    name   = "tag:Quality"
    values = [module.global.tags.quality]
  }
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.network.id]
  }
  provider = aws.network
}

data "aws_subnets" "public" {
  filter {
    name   = "tag:Connectivity"
    values = ["public"]
  }
  filter {
    name   = "tag:Environment"
    values = [module.global.tags.environment]
  }
  filter {
    name   = "tag:Quality"
    values = [module.global.tags.quality]
  }
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.network.id]
  }
  provider = aws.network
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
