# Bootstrapping your Substrate-managed AWS organization

The time has come! This section will guide you through running the first several Substrate commands to configure your Substrate-managed AWS organization.

## Decide where you'll run Substrate commands

All Substrate commands should be run from the same directory. Many commands will read and write files there to save your input and pass configuration and state from one command to another.

The most natural options are, in no particular order:

* The root of the Git repository where you keep your Terraform code
* A subdirectory of a Git repository
* A new Git repository just for Substrate

(Git is by no means required - you're free to choose any version control software.)

You can always change your mind on this simply by moving `modules`, `root-modules`, and `substrate.*` into another directory or repository.

Decide and change to that directory before proceeding.

If you like, now or later, you can set `SUBSTRATE_ROOT` in your environment to a fully-qualified directory pathname. Substrate will change to this directory at the beginning of every command so you don't have to micromanage your working directory.

## Bootstrapping your AWS organization

Run:

```shell-session
substrate setup
```

When you run this program, you'll be prompted several times for input. As much as is possible, this input is all saved to the local filesystem and referenced on subsequent runs.

The first of these inputs is an access key from the root of your new AWS account. It is never saved to disk (for security reasons) so keep it handy until later in this guide when it you're told it's safe to delete it. To create a root access key:

1. Visit [https://console.aws.amazon.com/iam/home#/security\_credentials](https://console.aws.amazon.com/iam/home#/security\_credentials)
2. Scroll to the _Access keys_ section
3. Click **Create access key**
4. Check the “I understand...” box and click **Create access key**
5. Click **Show**
6. Save these values in your password manager for now

This program creates your AWS organization and the member accounts Substrate uses to manage that organization. It installs a basic Service Control Policy and configures all the IAM roles and policies you need to move around your organization in a controlled manner.

This program will also prepare to run Terraform in any account in your organization and indeed use that capability to configure your networks.

To do so, it will ask for the names of your environments. Environments typically have names like “development” and “production” — they identify a set of data and all the infrastructure that may access it. (An advanced feature, qualities, are names given to independent copies of your infrastructure _within an environment_ that make it possible to incrementally change AWS resources. `substrate setup` doesn't get into these at first and instead defines a single quality called “default”. See [domains, environments, and qualities](../ref/domains-environments-qualities.html) to learn more.)

It also asks which AWS regions you want to use. Your answers inform how it lays out your networks to strike a balance between security, reliability, ergonomics, and cost. If you're unsure, starting with one region is fine.

Substrate will provision NAT Gateways for your networks if you so choose, which will enable outbound IPv4 access to the Internet from your private subnets. The AWS-managed NAT Gateways will cost about $100 per month per region per environment/quality pair. Without them, private subnets will only have outbound IPv6 access to the Internet; this is very possibly a fine state as the only notable thing that servers tend to access that isn't available via IPv6 is GitHub.

Even if you want Substrate to provision NAT Gateways, we recommend you initially answer “no” when prompted about them. This is because provisioning NAT Gateways for two or more environments will require more Elastic IP addresses than AWS' default service quota allows. Substrate will programmatically raise the limit on the relevant AWS service quotas but it may take AWS staff hours or days to respond. Thus, we think it's best to come back later to request these quotas be raised. Once they are, you can change that “no” to a “yes” and Substrate will provision your NAT Gateways.

Move on when you're prompted to open the AWS Console and choose a DNS domain name as part of Substrate's integration with your identity provider.
