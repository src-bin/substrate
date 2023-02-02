output "tags" {
  value = {
    domain      = local.domain_environment_quality_region[0]
    environment = local.domain_environment_quality_region[1]
    quality     = local.domain_environment_quality_region[2]
  }
}
