# Addressing SOC 2 criteria with Substrate

Most Substrate users are already or soon will be operating a SOC 2 compliance program. The following [SOC 2 criteria](https://us.aicpa.org/content/dam/aicpa/interestareas/frc/assuranceadvisoryservices/downloadabledocuments/trust-services-criteria.pdf) are at least partially addressed merely by adopting Substrate. In some cases, reference controls are provided that you're free to adopt verbatim or adapt as you and your auditors see fit.

## CC6.1

**The entity implements logical access security software, infrastructure, and architectures over protected information assets to protect them from security events to meet the entity's objectives.**

Substrate requires you to use an OAuth OIDC identity provider to control access to AWS (where, presumably, your “information assets” are hosted) to benefit from controls commonly applied to identity providers:

* Users have unique accounts for access to sensitive data, infrastructure, and tools.
* Access to sensitive data, infrastructure, and tools requires two-factor authentication.
* Users are granted access to sensitive data, infrastructure, and tools based on their role.

Provide evidence of the thoroughness of your integration between AWS and your identity provider by showing that there are no IAM users for any of your employees and demonstrating how to use Substrate tools to access AWS.

## CC6.2

**Prior to issuing system credentials and granting system access, the entity registers and authorizes new internal and external users whose access is administered by the entity. For those users whose access is administered by the entity, user system credentials are removed when user access is no longer authorized.**

Substrate integrates with your identity provider so that existing onboarding and offboarding processes serve to grant and revoke access to AWS without additional steps.

Provide evidence of existing onboarding and offboarding processes handling AWS by revoking a coworker's access and then granting it again.

## CC6.3

**The entity authorizes, modifies, or removes access to data, software, functions, and other protected information assets based on roles, responsibilities, or the system design and changes, giving consideration to the concepts of least privilege and segregation of duties, to meet the entity’s objectives.**

Substrate integrates with your identity provider so that existing change-of-role processes serve to modify access to AWS without additional steps.

Provide evidence of existing change-of-role processes handling AWS by changing your own or a coworker's role and then refreshing the Accounts page of your Intranet to observe the role changing.

## CC6.6

**The entity implements logical access security measures to protect against threats from sources outside its system boundaries.**

In a Substrate-managed AWS organization, all actors are considered to be outside the system's boundaries unless they're authorized by your identity provider. Once authorized, users can use AWS APIs or SSH (via the Instance Factory) to access sensitive data, infrastructure, and tools. Restricting production SSH access to 192.168.0.0/16 (the network used by your Substrate account) ensures that the Instance Factory is the only way in and access to the Instance Factory is controlled by your identity provider.

Controls to consider:

* Access to sensitive data, infrastructure, and tools is controlled by \[your identity provider].
* SSH access to sensitive infrastructure is restricted to \[role name(s)].

Provide evidence of these controls functioning by showing firewall rules restricting SSH access to 192.168.0.0/16, only your admin VPCs having CIDR prefixes within that, and only authorized users being able to use the Instance Factory.

## CC6.7

**The entity restricts the transmission, movement, and removal of information to authorized internal and external users and processes, and protects it during transmission, movement, or removal to meet the entity’s objectives.**

Internal users (read: your employees) are subject to whatever AWS IAM policies you attach to their role, which can serve to restrict both which AWS accounts they can access and which actions they can take there. Pay close attention, especially, to who can `s3:GetObject` on S3 buckets storing sensitive data. The IAM roles your services use as they broker access for external users (read: your customers) are subject to the same restrictions. In both cases, Substrate's use of multiple AWS accounts makes it easy to assert that services outside an account cannot access sensitive data within.

Also, concerning your Intranet: Substrate ensures your Intranet is always served over TLS with a valid certificate.

## CC8.1

**The entity authorizes, designs, develops or acquires, configures, documents, tests, approves, and implements changes to infrastructure, data, software, and procedures to meet its objectives.**

Substrate's insistence on using Terraform and multiple AWS accounts pays dividends here by creating separation between environments and qualities, making it easy to demonstrate that changes are tested before they reach (all of) production.

Controls to consider:

* Terraform enforces standard infrastructure configurations.
* Infrastructure changes are tested in development before being implemented in production.
* Infrastructure changes are implemented gradually in production. (Applicable if using multiple qualities and/or multiple regions.)
* The development and production environments are separated.
* Production data is not used or accessible in development.

Provide evidence of these controls functioning by showing that development and production are served by entirely different AWS accounts and that the VPCs for each environment are not peered with those from other environments.

## CC9.1

**The entity identifies, selects, and develops risk mitigation activities for risks arising from potential business disruptions.**

Substrate can help you architect your Terraform code to be multi-region-ready. Substrate also supports multiple environments which can facilitate disaster recovery exercises (or even the real thing).

## A1.2

**The entity authorizes, designs, develops or acquires, implements, operates, approves, maintains, and monitors environmental protections, software, data backup processes, and recovery infrastructure to meet its objectives.**

So long as you define your infrastructure using `module.substrate.private_subnet_ids` and `module.substrate.public_subnet_ids`, you can be guaranteed to always be operating in three availability zones, even if you're only using one region.

Control to consider: Infrastructure is provisioned across three availability zones.

Provide evidence of this control functioning by showing your Terraform code, actual resources (ECS or EKS clusters, autoscaling groups, database clusters, etc.) in the AWS Console, and the [documentation for `module.substrate.private_subnet_ids` and `module.substrate.public_subnet_ids`](../mgmt/writing-terraform-code.html).
