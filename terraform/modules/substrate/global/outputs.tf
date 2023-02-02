output "tags" {
  value = {
    domain      = local.tags[0]
    environment = local.tags[local.tags[0] == "admin" ? 0 : 1]
    quality     = local.tags[local.tags[0] == "admin" ? 1 : 2]
  }
}
