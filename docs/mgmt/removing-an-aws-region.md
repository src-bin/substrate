# Removing an AWS region

You may at some point decide you want to abandon an AWS region you're using. Perhaps your traffic has shifted, perhaps your architecture's changed, or perhaps it's for some other reason. No matter, here's how you remove a region from Substrate's management and your root Terraform modules.

## Ensure service continuity and data durability

First and foremost, if you're going to remove a region and all the infrastructure there, ensure you're not going to cause an outage or data loss when you do. Ensure that services will continue to operate, either by having sufficient capacity elsewhere or by proactively draining traffic from the region that's about to be removed. Ensure any data stored in the region is available in at least one but preferably two other regions.

## Remove the region from Substrate

Run `substrate setup --fully-interactive`. When prompted if your region list is accurate, respond “no”, and remove the region from your list of regions. When this command exits, you'll have a great many directories in `root-modules` that reference the region which will no longer be managed by any Substrate tools.

## Destroy admin and service infrastructure in the region

`find root-modules -name <region>` to list all the root modules that run in the region. Starting with your admin and service accounts, run `terraform destroy` in each one and then `rm -f -r` that directory.

## Destroy network peering relationships

`find root-modules/network/peering -name <region>` to list all the root modules that manage network peering relationships that involve the region. Run `terraform destroy` and `rm -f -r` for each one, just as you did for service accounts before.

## Destroy deploy buckets and the networks themselves

`find root-modules -name <region>` again and run `terraform destroy` and `rm -f -r` for everything that's left. It's likely that you'll have to destroy the networks twice to fully resolve Terraform's dependencies.
