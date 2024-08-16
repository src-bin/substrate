# Adding a domain

Domains are a mechanism for protecting one service (or group of services) from others. You may create as many as you like. Creation and subsequent updates are simple: Run `substrate account create --domain <domain> --environment <environment>` with the name of your (new) domain and a declared environment. This will create a new AWS account in your organization, add it to `substrate.accounts.txt`, and create all the necessary IAM roles to allow administrators to access the account.

If not immediately, you'll eventually create this domain in all of your environment/quality pairs to enable a complete progression from e.g. development through production.

See [domains, environments, and qualities](../ref/domains-environments-qualities.html) for more discussion of these fundamental Substrate concepts.

## Generated Terraform modules

All accounts with a given domain, across all environments and qualities, will be generated with Terraform code that references a generated Terraform module named the same as the domain. This is where you should put the vast majority of Terraform resources, possibly parameterized by `module.substrate.tags.domain`, `module.substrate.tags.environment`, and `module.substrate.tags.quality` as well as `module.substrate.public_subnet_ids` and `module.substrate.private_subnet_ids`.

If you choose to add `variable` stanzas to that module, we recommend that you do not set a `default` for those variables; this will force you to consider the appropriate values when creating this domain in different environments and/or qualities.

Run Terraform with `substrate account update`, `substrate terraform`, or `terraform` (directly in a root module directory).
