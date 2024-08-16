# Multi-region strategy

Substrate is multi-region through and through. It keeps a strict separation between [global and regional Terraform modules](global-and-regional-terraform-modules.md), manages CIDR prefixes and peers VPCs, and helps you implement changes region-by-region. (It's worth noting explicitly, too, that all of this works just fine when using only one region.)

`substrate setup` allows you to select which regions you're using (see [adding](../mgmt/adding-an-aws-region.md) and [removing](../mgmt/removing-an-aws-region.md) documentation). But which should you choose?

First things first: You should not simply select every region. For one thing, it won't work due to the hard limits on VPC peering. (Peering multiple domains, each with multiple qualities across every AWS region is possible using Transit Gateway but that is expensive, less reliable, and not currently supported by Substrate.) For another, it will make applying Terraform changes terribly slow. Finally, if you're using NAT Gateways, the costs will pile up to the tune of at least $100 per environment per region per month.

Your first region should be geographically close to a large portion of your customers and in a jurisdiction where they'll accept their data being stored. We hope you'll choose one of the regions AWS guarantees to cover with [renewable energy or at least credits/offsets](https://sustainability.aboutamazon.com/environment/the-cloud?energyType=true):

* ca-central-1 in Canada
* eu-central-1 in Frankfurt
* eu-west-1 in Ireland
* us-gov-west-1 in the northwestern United States
* us-west-2 in Oregon

If you're building to maximize reliability and can stomach the engineering cost of building a multi-region system, choose additional regions to suit your durability, geographic, and latency requirements. That may mean adding more regions on the same continent as the one you chose first or spreading yourself all around the world.

By default, when you create service accounts, the generated Terraform code is going to instantiate the domain's regional Terraform module in every region. You can delete some of these `module` stanzas from `root-modules/`_`domain`_`/`_`environment`_`/`_`quality`_`/`_`region`_`/main.tf` if you so choose, to the effect of having some regions that run less of your infrastructure than others. This allows, among other things, you to create your own notion of “core” and “edge” regions. Substrate will not re-generate the `module` stanzas you remove.

Underlying all the advice above is this: Do as little in us-east-1 as you possibly can. Ideally, don't even configure us-east-1 as one of your regions and instead only use it via the special `aws.us-east-1` provider in your global Terraform modules and only then for managing pseudo-global resources from e.g. ACM or Lambda@Edge that must be managed in us-east-1.
