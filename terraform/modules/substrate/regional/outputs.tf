output "tags" {
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
