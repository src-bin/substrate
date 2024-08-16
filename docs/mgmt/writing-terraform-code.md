# Writing Terraform code

## Writing Terraform code with an assist from Substrate

Substrate generates a directory structure for your Terraform code, remote state file configurations, provider configurations, and `module` blocks so you can get started right away writing Terraform code that matters to your business.

### Directory structure

**tl;dr: Write code in the `modules` subdirectories Substrate creates for each domain. Run Terraform with `substrate account update`, `substrate terraform`, or simply by running Terraform in the appropriate `root-modules` subdirectories.

Suppose you're responsible for a service called _example_ that you run in _staging_ as _alpha_-quality and in _production_ as _beta_- and _gamma_-quality. You will create those environments and qualities interactively with `substrate setup` and will then run the following commands to create all your service accounts:

* `substrate account create --domain example --environment staging --quality alpha`
* `substrate account create --domain example --environment production --quality beta`
* `substrate account create --domain example --environment production --quality gamma`

These will create the following directory trees, which you will should commit to version control:

* `modules`
  * `common`
    * `global`
    * `regional`
  * `example`
    * `global`
    * `regional`
* `root-modules`
  * `example`
    * `staging`
      * `alpha`
        * `global`
        * `us-east-2`
        * `us-west-2`
    * `production`
      * `beta`
        * `global`
        * `us-east-2`
        * `us-west-2`
      * `gamma`
        * `global`
        * `us-east-2`
        * `us-west-2`

What do you do next? And where do you do it?

The vast majority of your work should happen in your domain's Terraform modules. In this _example_ domain those are `modules/example/global` and `modules/example/regional`. Put global resources like IAM roles and Route 53 records in `modules/example/global`. Put regional resources like autoscaling groups and EKS clusters in `modules/example/regional`. (Substrate has generated all the `module` blocks necessary to instantiate these modules with the right Terraform providers.)

You selected some number of AWS regions when you configured your network but you may not want to run all your infrastructure in all those regions all the time (if for no other reason than cost control). You may edit `root-modules/*/*/*/*/main.tf` to customize which of your selected regions are actually in use. By default, all your selected regions are in use. If you don't want to provision your infrastructure in any of them, simply comment out the resources in `root-modules/*/*/*/*/main.tf` (substituting your domains, environments, qualities, and regions as desired).

Every domain module automatically instantiates a common module, too. Here, `modules/example/global` instantiates `modules/common/global` and `modules/example/regional` instantiates `modules/common/regional`. These common modules, following the now-familiar pattern of [global and regional Terraform modules](../ref/global-and-regional-terraform-modules.md), are for resources that should exist in every service account. Note that the common modules are not instantiated in the Substrate, audit, deploy, management, or network accounts.

### Testing and deploying Terraform changes

It's no accident that `modules/example/global` and `modules/example/regional` are referenced by the root Terraform modules for every environment and quality in the domain. These afford you multiple opportunities to implement changes in pre-_production_ and partial-_production_ in order to catch more bugs and failures before they impact all your capacity and all your customers. Continuing from the example above, here is the complete lifecycle of a Terraform change in the _example_ domain:

1. Change e.g. an EC2 launch template in `modules/example/regional/main.tf`
2. Commit, push, get a code review, and merge into the main branch
3. `substrate account update --domain example --environment staging --quality alpha` and verify your changes
4. `substrate account update --domain example --environment production --quality beta` and verify your changes, either at the end or at each pause between regions
5. `substrate account update --domain example --environment production --quality gamma` and verify your changes, either at the end or at each pause between regions

## Referencing Substrate parameters in Terraform

As you write your own Terraform modules, you're certainly going to want to parameterize them in the same ways Substrate helps you parameterize your AWS accounts. Plus, you're also going to need a network, and Substrate's already created some and shared the right one with every service account to make it easy, secure, and cost-effective to build new things.

`substrate account create` automatically creates global and regional Terraform modules for your domain in `modules/<domain>`. Those modules include a reference to `modules/substrate` which provides the following helpful context:

* `module.substrate.tags.domain`: The domain of this AWS account, from the tags on the account itself.
* `module.substrate.tags.environment`: The environment of this AWS account, from the tags on the account itself.
* `module.substrate.tags.quality`: The quality of this AWS account, from the tags on the account itself.
* `module.substrate.cidr_prefix`: The CIDR prefix of this environment/quality's shared VPC.
* `module.substrate.private_subnet_ids`: A _set_ of three private subnet IDs in this environment/quality's shared VPC.
* `module.substrate.public_subnet_ids`: A _set_ of three public subnet IDs in this environment/quality's shared VPC. (This set is empty in your Substrate account, which only has public networks because bastion/jump hosts need public IP addresses.)
* `module.substrate.vpc_id`: The ID of this environment/quality's shared VPC.
