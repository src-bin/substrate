# Technology choices

Very little about your use of AWS is constrained by Substrate. Beyond a few central assumptions, your choices are as endless as AWS makes them. This is a double-edged sword. Part of the value of Substrate is in guiding you through the endless maze that must be navigated just to _start_ using AWS in a scalable manner.

Substrate stays modest in the constraints it imposes:

* It wants you to use AWS accounts to create separation between domains (services or groups of services) and between environments (data sets and the infrastructure that store and process them).
* It wants you to use shared VPCs (that it manages) to simplify network layout and minimize transit costs. (Note well, though, that this is not required. You're welcome to create more VPCs and wire them together with VPC Peering or even PrivateLink.)
* It wants you to be prepared to deploy to multiple regions even if you don't choose to do so immediately or at all times.
* It wants you to use Terraform to define the vast majority of your infrastructure and it wants you to organize your Terraform resources into global and regional modules to preserve your ability to run in multiple regions. See [global and regional Terraform modules](global-and-regional-terraform-modules.md) for more information.

That leaves weighty choices like service architecture, storage and compute technologies, multi-tenancy, traffic routing, and more entirely up to you. These can still be overwhelming choices. Some advice follows.

## Storage technologies

The particular storage technology you choose makes absolutely no difference but you should absolutely count on hosting many instances of your chosen storage technologies, with at least one in every domain account in every environment. Service-oriented architecture is (partially) about encapsulation, after all, and the services in a domain should absolutely be encapsulating their data.

## Compute technologies

As with storage technologies, your choice of compute technology is your own except that if you choose to use container orchestration frameworks like ECS or Kubernetes you should operate e.g. one cluster per AWS account. This ensures that the failure domain in case of cluster failure is the same as in case of application failure. ECS and Kubernetes users may be (rationally, justifiably) tempted to define slightly larger domains than they otherwise would in order to take advantage of some economies of scale by having larger clusters where more bin-packing efficiency can be had.

## DNS, load balancers, and accepting traffic from the Internet

It seems like Substrate should express an opinion on how DNS domains and load balancers are organized. The truth, though, is that these decisions are more tightly tied to how traffic enters your network than how that network is laid out.

If you have a single service that “fronts” all your other services then buying your DNS names and hosting your load balancers in domain accounts makes sense.

On the other hand, if you use a single load balancer to route traffic to many services spread across more than one domain then buying DNS domains and hosting the load balancer in a dedicated DNS account or even the network account makes sense.

As for the practical matter of buying DNS domain names, here's what we recommend:

1. Buy the DNS domain name using the Route 53 console in the appropriate AWS account for the domain, environment, and quality (e.g. buying [example-staging.com](http://example-staging.com) in a staging account)
2. Use a Terraform data source to refer to the hosted zone created when you buy the domain
3. Manage DNS records in Terraform as usual

## Serverless

If you want to go to the extreme, you can use Substrate in a purely serverless environment by simply not using the VPCs Substrate manages and shares into your domain accounts.

If you take this path, you should ensure you've answered “no” to Substrate managing NAT Gateways in your networks, as they cost about $100 per month per region per environment/quality pair.

## Secrets

Where to store your secrets? If you choose AWS Secrets Manager, Hashicorp Vault, something else that runs in your organization (i.e. something that isn't a SaaS product), you'll need to decide which AWS accounts host these services.

I think the most important property of any secret management strategy is that it makes storing a secret the right way easier than storing one the wrong way i.e. committed alongside your source code. For my money, the solution with the lowest barrier to entry is AWS Secrets Manager in whatever account needs to access the secrets. This makes the requisite IAM policies straightforward to write, access restricted pretty well by default, and adding new secrets as easy as I can imagine. (See the `aws-secrets-manager` tool bundled with Substrate for an example of how easy putting secrets into AWS Secrets Manager can be.)
