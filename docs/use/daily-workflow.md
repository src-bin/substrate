# Daily workflow

Substrate is not here to dictate all aspects of your daily workflow, nor is Substrate in the business of influencing or limiting your tool choices. Substrate is all about removing all the hassles that come with having lots of AWS accounts, the one true unit of isolation in AWS, so that you can reap all the security, reliability, and compliance benefits of isolating your workloads in their own AWS accounts.

As such, Substrate introduces some extra tools to your daily workflow, mostly concerning accessing AWS and moving between AWS accounts.

## Prerequisites

You should either be the person who bootstrapped Substrate at your company or have followed the [getting started](getting-started.md) guide already. It's a great idea to make sure `SUBSTRATE_ROOT` is set in your environment to the fully-qualified pathname where you've cloned your Substrate repository.

## Get AWS credentials from your Credential Factory

At the beginning of each working day, you'll want to refresh your AWS credentials, since each set only lasts 12 hours:

```shell-session
eval $(substrate credentials)
```

## Assume roles to move between AWS accounts

Learn what AWS accounts exist in your organization and how they're tagged by looking in `substrate.accounts.txt` or running `substrate account list`.

You can run a one-off command in one of those accounts (and of course the one-off command doesn't have to be `aws ec2 describe-instances`):

```shell-session
substrate assume-role --domain <domain> --environment <environment> aws ec2 describe-instances
```

([Domains, environments, and qualities](../ref/domains-environments-qualities.md) are how Substrate organizes AWS accounts.)

Or you can move your whole terminal session into another account:

```shell-session
eval $(substrate assume-role --domain <domain> --environment <environment>)
```

And return from whence you came when you've wrapped up your work in that service account:

```shell-session
unassume-role
```

In all of these situations, the account boundary serves as a critical isolating safety feature, ensuring exploratory changes in development can't impact production or emergency operational changes to one production service can't impact others.

## Create and update AWS accounts and IAM roles

In a Substrate-managed AWS organization you'll create and update accounts and IAM roles (always confident that Substrate will find them if they already exist).

When you create (or update) an account, Substrate will ensure the account and its basic IAM roles are in good working order and then run the various Terraform root modules associated with the account (one for global resources and another for each region; see [global and regional Terraform modules](../ref/global-and-regional-terraform-modules.md) for more).

Try it for yourself, using the domain and environment from a service account you find listed in `substrate.accounts.txt`:

```shell-session
substrate account update --domain <domain> --environment <environment>
```

Whenever necessary, you can create a new account for a new purpose:

```shell-session
substrate account create --domain <domain> --environment <environment>
```

Substrate manages an all-powerful Administrator role and a limited read-only Auditor role by default. But you're probably going to want to create custom IAM roles to complement the isolation you create by having lots of AWS accounts. Substrate manages IAM roles for cross-account access better than anything else around.

```shell-session
substrate role create --role <RoleName> [account selection flags] [assume-role policy flags] [policy attachment flags]
```

There are a lot of options, though, so consult the documentation on [adding custom IAM roles for humans or services](../mgmt/custom-iam-roles.md) to get the complete picture.

## Plan and apply Terraform changes

Substrate gives you production-ready Terraform infrastructure for all your AWS accounts with a module structure that enhances the isolation provided by your many AWS accounts and locked, remote state files. It strives to make [writing Terraform code](../mgmt/writing-terraform-code.md) a straightforward exercise free of yak-shaving.

And while `substrate account update` does in fact plan and/or apply Terraform changes in all an account's root modules in a predictable order, iterating on your works-in-progress deserve a faster feedback loop:

```shell-session
substrate terraform --domain <domain> --environment <environment> --region <region> plan
substrate terraform --domain <domain> --environment <environment> --region <region> apply
```

## Launch an EC2 instance

In addition to brokering AWS credentials via your identity provider, your Intranet also includes the Instance Factory that can provision personal, temporary EC2 instances in your Substrate account for use as jump boxen or development environments.

To provision your own, visit your Intranet in your web browser, click Instance Factory, choose a region, and choose an instance type.

## Login to the AWS Console

Sometimes the fastest way to understand what's going on is to use the AWS Console. With lots of AWS accounts, though, this can be tricky. Your Intranet's Accounts page provides direct links into the AWS Console as your assigned IAM role or as the limited read-only Auditor role.

Or, if you're starting from a terminal, you can open the AWS Console by using the `--console` option to `substrate assume-role` (with all the rest of the options just like you'd normally provide).
