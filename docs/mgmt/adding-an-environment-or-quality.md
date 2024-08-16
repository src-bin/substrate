# Adding an environment or quality

Environments and qualities create isolation between groups of AWS accounts to allow you to keep development separate from production yet also incrementally implement changes to AWS resources. Most customers begin their time using Substrate with a couple of environments and a single quality and are eventually motivated to add more of each.

Add more environments to create entirely disjoint data sets with all the supporting infrastructure. You might want these for additional testing, disaster recovery, or even to isolate each of your customers from the others.

Add more qualities to create the opportunity to incrementally change load balancers, autoscaling groups, security groups, DNS zones, IAM roles, and other AWS resources that are typically difficult to incrementally change.

See [domains, environments, and qualities](../ref/domains-environments-qualities.md) for more discussion of these fundamental Substrate concepts.

To add an environment, a quality, or both, follow these steps:

1. Run `substrate setup --fully-interactive`
2. If you wish to add an environment, answer “no” when asked if your list of environments is correct and follow the prompts
3. If you wish to add a quality, answer “no” when asked if your list of qualities is correct and follow the prompts
4. Answer “no” when asked if the valid environment and quality pairs are correct
5. Then answer “yes” and “no” accordingly to allow whatever new combinations of environment and quality you wish

Be warned: If your additions cause sufficiently many NAT Gateways to be created, the tools will open support cases on your behalf to have service quotas raised. AWS is often not the most accommodating with these requests. If AWS gives you a hard time it is likely because they don't want to give you additional Elastic IPs. **Tell them you're trying to create VPCs to share to all your AWS accounts, each with three private subnets and zonal NAT Gateways, which is exactly how AWS wants you to design your networks.** These support cases can unfortunately take hours or even days to be resolved; they're usually fastest to respond if you request a chat or phone response. By the way, if `substrate setup` exits early, it may simply be re-run; it will find the existing request to increase your quota and continue waiting for it to be resolved.

Once you've added an environment and/or quality, you'll need to use `substrate account create` to add each of your domains to the new environment/quality pair(s). See [adding a domain](adding-a-domain.md) for more on that step.
