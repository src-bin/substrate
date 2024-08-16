# Domains, environments, and qualities

Substrate leans heavily on three concepts in order to improve the reliability of the overall system by minimizing the blast radius of changes.

They're usually referred to in alphabetical order - domain, environment, and quality - but are presented here in a progression more suitable to readers new to Substrate.

## Environments

Environments identify a set of data and the infrastructure that stores and processes it. (After all, what distinguishes your production environment from development, staging, or another? Production _data_; your customers' _real, business-critical data_.) An environment's primary purpose is to protect its data against access from other environments.

Use multiple environments to protect your customers’ data from code that hasn’t been tested thoroughly in pre-production environments.

An AWS account in your organization is a member of exactly one environment and can only access the networks assigned to that environment.

### Examples

Organizations typically define environments like _development_, _staging_, and _production_ though the names and number is entirely up to you. Add more environments to, for example, support more different kinds of testing with greater parallelism.

## Qualities

Highly reliable services almost always implement changes gradually to give their operators a chance to detect and mitigate failures when the impact is small. Qualities help make incremental change possible for AWS resources that would otherwise be difficult to incrementally change like load balancers, autoscaling groups, security groups, DNS zones, IAM roles, and more, even within a single service.

Use multiple qualities to protect any one service from changes that affect that whole service immediately.

An AWS account in your organization is associated with exactly one quality but can access and use resources in AWS account of any quality so long as they share the same environment.

### Examples

Suppose your organization defined the qualities _alpha_, _beta_, and _gamma_ (which is one set of qualities that Substrate recommends). You could run 1% of your production environment in your alpha accounts, 9% in your beta accounts, and the remaining 90% in your gamma accounts. This isn't as smooth as routing a slowly increasing percentage of traffic to your new software as it's being deployed (and you should strongly consider doing that, too, eventually) but this strategy helps you incrementally change any AWS resource.

You could also decide to name your qualities _blue_ and _green_ and swing traffic back and forth between them. The slight disadvantage to this architecture is that the one that's not receiving any traffic is not, at that moment, proving that its configuration is functional and thus the first trickle of traffic that comes to it when you start to swing back to it is slightly higher risk.

Or you could decide name your only quality _default_. (Most folks using Substrate do this at first.) Later, when you need it, you can add a _canary_ quality that your deploy to first and that takes a small fraction of your traffic. You might also add an _enterprise_ or _proven_ quality that you deploy to last where your highest-paying or most-risk-averse customers are routed.

## Domains

Domains are groups of one or more software services that form an isolated failure domain (pun very much intended). The software may be that which you've written yourself, hosted in any serverless or serverful manner, an AWS-managed service, something you bought from the AWS Marketplace, or a SaaS product that manages infrastructure in one of your AWS accounts.

Use multiple domains to protect services in any one domain from changes in all other domains. Group services into a single domain when they're tightly coupled, share the same level of access to AWS or other services, or are developed and deployed together.

An AWS account in your organization is associated with exactly one domain but can access network services in any AWS account that shares its environment. There may be multiple AWS accounts within a domain, each in a different environment or of a different quality.

### Example

It is intended that every AWS account that shares the same domain also shares the same Terraform codebase. That codebase progresses through environments and qualities as changes are deployed. For example, consider the domain _example_ which exists in the following environments and qualities:

* Domain: _example_, Environment: _staging_, Quality: _alpha_
* Domain: _example_, Environment: _production_, Quality: _beta_
* Domain: _example_, Environment: _production_, Quality: _gamma_

They all refer back to the same modules, parameterized according to their domain, environment, and quality plus the appropriate VPC and subnet IDs. The difference between them, then, is _when_ changes are deployed.
