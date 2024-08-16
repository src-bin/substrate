# Upgrading Terraform

Since Substrate 2023.09, the version of Terraform that Substrate helps you install is controlled by the `terraform.version` file in your Substrate repository. The contents of this file must be a complete Terraform version number like "1.5.6" (and, per UNIX convention, a trailing newline is encouraged).

Upgrade Terraform as follows:

1. Change the contents of `terraform.version` to specify the desired version.
2. Commit that change to version control.
3. Run `substrate terraform install` (and mention to your teammates that they'll be prompted to do the same when their local version mismatches).
4. Run `substrate setup` and `substrate account update` in all your service accounts. The one-command way to do this is: `sh <(substrate account list --format shell)`

Likewise, the version of the Terraform AWS provider that Substrate will include as a constraint in generated Terraform root modules is controlled by the `terraform-aws.version-constraint` file in your Substrate repository. The contents of this file may be any Terraform provider version constraint but is most often the `~>` operator followed by the major and minor components of the minimum provider version number, like "~> 4.67" (with a trailing newline as mentioned above).

Upgrade the Terraform AWS provider as follows:

1. Change the contents of `terraform-aws.version-constraint` to specify the desired minimum version.
2. Commit that change to version control.
3. Run `substrate setup` and `substrate account update` in all your service accounts. The one-command way to do this is: `sh <(substrate account list --format shell)`
