locals {
  pathname       = abspath(path.root)
  pathname_parts = split("/", local.pathname)
  tags = slice(
    local.pathname_parts,
    length(local.pathname_parts) - index(reverse(local.pathname_parts), "root-modules"),
    length(local.pathname_parts),
  )
}
