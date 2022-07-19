output "cidr_prefix" {
  value = data.aws_vpc.network.cidr_block
}

output "private_subnet_ids" {
  value = module.global.tags.environment == "admin" ? [] : data.aws_subnets.private[0].ids
}

output "public_subnet_ids" {
  value = data.aws_subnets.public.ids
}

output "tags" {
  value = module.global.tags
}

output "vpc_id" {
  value = data.aws_vpc.network.id
}
