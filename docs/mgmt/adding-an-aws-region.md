# Adding an AWS region

Substrate is designed to accommodate multi-region infrastructures from day one and is ready at a moment's notice if you start with one region but decide suddenly to add more.

As a prerequisite to adding a new region, it's critical that you follow the advice on [global and regional Terraform modules](../ref/global-and-regional-terraform-modules.html) so that additional regional modules don't introduce conflicts during your Terraform runs.

Regions are added by `substrate setup`. To add a new one (or two), simply respond as follows to its prompts:

1. Run `substrate setup --fully-interactive`
2. Answer “no” when asked if your list of regions is correct
3. Add your new one in the text editor that was opened for you, paying attention to the order you want your qualities presented
4. Save and exit the text editor
5. If your changes cause sufficiently many VPCs to be created, the tools will open support cases on your behalf to have service quotas for VPCs and related resources raised
   * If AWS gives you a hard time it is likely because they don't want to give you additional Elastic IPs; tell them you're trying to create VPCs to share to all your AWS accounts with three private subnets and zonal NAT Gateways and they'll hopefully let you get on with your life but if they don't, feel free to escalate to Source & Binary
   * These support cases sometimes take hours or even days to be resolved
   * If the tool exits or is interrupted it may simply be re-run
6. When `substrate setup` exits successfully, your network is ready for use in the added region(s)
7. Run `substrate account update` for all your domain, environment, quality accounts
