# Root Terraform modules

Terraform modules are just directories. You've learned about how Substrate organizes [global and regional Terraform modules](global-and-regional-terraform-modules.html) already. Substrate organizes root Terraform modules similarly, within the `root-modules` directory tree, with some having a leaf directory name of `global` and others having a leaf directory name of a particular AWS region.

Root Terraform modules aren't just different because their regional modules name actual regions, though. Root Terraform modules include extra configuration that declares how each and every `provider` is configured — which role to assume in which AWS account — and how Terraform state is stored. (Substrate always arranges to store Terraform state in S3 and DynamoDB in your deploy account, which it accesses using the TerraformStateManager role.)

Root Terraform modules are where you actually invoke Terraform commands. You can do so in one of three ways (and using any Terraform subcommand, not just `plan`):

* `cd` into the root Terraform module and run `terraform plan`
* `terraform -chdir=`_`root-modules/...`_` ``plan`
* `make -C`_`root-modules/...`_` ``plan` (for compatibility with Substrate before Terraform introduced `-chdir`)
