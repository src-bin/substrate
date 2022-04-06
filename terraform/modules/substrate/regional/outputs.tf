output "tags" {
  value = module.global.tags
}

output "private_subnet_ids" {
  value = module.global.tags.environment == "admin" ? [] : data.aws_subnets.private[0].ids
}

output "public_subnet_ids" {
  value = data.aws_subnets.public.ids
}

output "vpc_id" {
  value = data.aws_vpc.network.id
}
