# Substrate documentation

New to [Substrate](https://substrate.tools)? Here's the deal: In AWS, the one true unit of isolation is the AWS account but isolating all your environments and services in their own AWS accounts can be tedious. **Substrate removes all the hassles that come with having lots of AWS accounts** - access, navigation, IAM roles and permissions, networking, and more - so you can reap all the security, reliability, and compliance benefits of true isolation between your AWS workloads.

If you're the first person at your company to pick up Substrate, begin by [bootstrapping your AWS organization](bootstrapping/overview.html).

New users at companies already using Substrate can jump straight to [getting started](use/getting-started.html) with Substrate and learn about the [daily workflow](use/daily-workflow.html) Substrate encourages.

Once you're up and running, this site has all your references for common access and management tasks, resources for your first (or fourteenth) SOC 2 audit, architectures that get the most of of Substrate and AWS, and more.

This Documentation is also available on the [GitHub](https://github.com/src-bin/substrate-docs), so feel free to open an [issue](https://github.com/src-bin/substrate-docs/issues) or a [pull request](https://github.com/src-bin/substrate-docs/pulls) if you find any problems or want to suggest improvements.

## Table of contents

* [Release notes](releases.html)

### Bootstrapping your Substrate-managed AWS organization <a href="#bootstrapping" id="bootstrapping"></a>

* [Overview](bootstrapping/overview.html)
* [Opening a fresh AWS account](bootstrapping/opening-a-fresh-aws-account.html)
* [Installing Substrate and Terraform](bootstrapping/installing.html)
* [Configuring Substrate shell completion](bootstrapping/shell-completion.html)
* [Bootstrapping your Substrate-managed AWS organization](bootstrapping/bootstrapping.html)
* [Integrating your identity provider to control access to AWS](bootstrapping/integrating-your-identity-provider/README.html)
  * [Integrating your Azure AD identity provider](bootstrapping/integrating-your-identity-provider/azure-ad.html)
  * [Integrating your Google identity provider](bootstrapping/integrating-your-identity-provider/google.html)
  * [Integrating your Okta identity provider](bootstrapping/integrating-your-identity-provider/okta.html)
* [Finishing up in your management account](bootstrapping/finishing.html)
* [Configuring CloudTrail](bootstrapping/cloudtrail.html)
* [Integrating your original AWS account(s)](bootstrapping/integrating-your-original-aws-account.html)

### Using Substrate <a href="#use" id="use"></a>

* [Getting started (after someone else has bootstrapped Substrate)](use/getting-started.html)
* [Daily workflow](use/daily-workflow.html)
* [Upgrading Substrate](use/upgrading.html)
* [Upgrading Terraform](use/upgrading-terraform.html)

### Accessing and navigating AWS <a href="#access" id="access"></a>

* [Accessing AWS in your terminal](access/accessing-aws-in-your-terminal.html)
* [Accessing the AWS Console](access/accessing-the-aws-console.html)
* [Moving between AWS accounts](access/moving-between-aws-accounts.html)
* [Using AWS CLI profiles](access/aws-cli-profiles.html)
* [Jumping into private networks](access/jumping-into-private-networks.html)
* [Enumerating all your AWS accounts](access/enumerating-all-your-aws-accounts.html)
* [Enumerating all your root Terraform modules](access/enumerating-all-your-root-terraform-modules.html)
* [Enumerating all your custom AWS IAM roles](access/enumerating-all-your-aws-iam-roles.html)
* [Cost management](access/cost-management.html)
* [Deep-linking into the AWS Console](access/deep-linking-into-the-aws-console.html)

### Managing AWS accounts, roles, and resources <a href="#mgmt" id="mgmt"></a>

* [Managing your infrastructure in service accounts](mgmt/service-accounts.html)
* [Adding a domain](mgmt/adding-a-domain.html)
* [Adding an environment or quality](mgmt/adding-an-environment-or-quality.html)
* [Adding custom IAM roles for humans or services](mgmt/custom-iam-roles.html)
* [Onboarding users](mgmt/onboarding-users.html)
* [Offboarding users](mgmt/offboarding-users.html)
* [Allowing third parties to access your AWS organization](mgmt/allowing-third-parties-to-access-your-aws-organization.html)
* [Adding administrators to your AWS organization](mgmt/adding-administrators.html)
* [Subscribing to AWS support plans](mgmt/aws-support.html)
* [Removing an AWS account from your organization](mgmt/removing-an-aws-account-from-your-organization.html)
* [Closing an AWS account](mgmt/closing-an-aws-account.html)
* [Adding an AWS region](mgmt/adding-an-aws-region.html)
* [Removing an AWS region](mgmt/removing-an-aws-region.html)
* [Using Amazon EC2 when IMDSv2 is required](mgmt/ec2-imdsv2.html)
* [Customizing EC2 instances from the Instance Factory](mgmt/customizing-instance-factory.html)
* [Writing Terraform code](mgmt/writing-terraform-code.html)
* [Additional Terraform providers](mgmt/additional-terraform-providers.html)
* [Deploying software](mgmt/deploying-software.html)
* [Protecting internal tools](mgmt/protecting-internal-tools.html)

### Compliance

* [Addressing SOC 2 criteria with Substrate](compliance/addressing-soc-2-criteria-with-substrate.html)
* [Auditing your Substrate-managed AWS organization](compliance/auditing.html)

### Architectural reference <a href="#ref" id="ref"></a>

* [Accounts in a Substrate-managed AWS organization](ref/accounts.html)
* [Diagram of a Substrate-managed AWS organization](ref/diagram-substrate-managed-aws-organization.html)
* [Domains, environments, and qualities](ref/domains-environments-qualities.html)
* [Global and regional Terraform modules](ref/global-and-regional-terraform-modules.html)
* [Root Terraform modules](ref/root-terraform-modules.html)
* [Substrate filesystem hierarchy](ref/substrate-filesystem-hierarchy.html)
* [Networking](ref/networking.html)
* [Diagram of a multi-quality, multi-region service](ref/diagram-multi-quality-multi-region-service.html)
* [Multi-region strategy](ref/multi-region-strategy.html)
* [Technology choices](ref/technology-choices.html)
* [Multi-tenancy](ref/multi-tenancy.html)
* [Deciding where to host internal tools](ref/internal-tools.html)
* [Telemetry in Substrate](ref/telemetry.html)
* [Changes to Substrate commands in 2024.01](ref/command-changes-2024.01.html)

### Runbooks for emergency and once-in-a-blue-moon operations <a href="#runbooks" id="runbooks"></a>

* [Changing identity providers](runbooks/changing-identity-providers.html)
* [Sharing CloudWatch data between accounts](runbooks/cloudwatch-sharing.html)
* [Regaining access in case the Credential and Instance Factories are broken](runbooks/regaining-access.html)
* [Debugging Substrate](runbooks/debugging.html)
* [AWS IAM Identity Center](runbooks/aws-iam-identity-center.html)

### Meta

* [Typographical conventions](meta/typography.html)
