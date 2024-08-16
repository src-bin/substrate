# Getting started (after someone else has bootstrapped Substrate)

This guide is meant to help folks get started using Substrate at a company that has already bootstrapped Substrate. If you're the first person at your company to pick up Substrate, you'll want to start by [bootstrapping your AWS organization](../bootstrapping/overview.md). If you're joining a party already in progress, read on.

## Installing Substrate and Terraform

Get a Substrate download URL from your coworkers. Take care to select the appropriate operating system and CPU architecture. Substrate releases for MacOS (“`darwin`”) and Linux on 64-bit x86 (“`amd64`”) and ARM (“`arm64`”).

1. Download, either by clicking the appropriate link on your Intranet's Substrate page or directly: `curl -O https://src-bin.com/substrate-<version>-<OS>-<ARCH>.tar.gz`
2. Extract: `tar xf substrate-<version>-<OS>-<ARCH>.tar.gz`
3. Install: `cp substrate-<version>-<OS>-<ARCH>/bin/substrate ~/bin` (substituting your preferred writable directory in your PATH for “`~/bin`”, if you wish)

Install the version of Terraform your organization requires: `substrate terraform install`

No Substrate installation is truly complete without shell completion, which is provided for Bash, Fish, and Z shell (and any other shell with Bash-compatible completion). Add the appropriate configuration to your `~/.profile` and run `. ~/.profile`:

```shell
. <(substrate shell-completion)
```

## Clone your company's Substrate repository

Substrate asked for several configuration files to be stored in version control during bootstrapping. You need to clone this repository and have access to these files in order to use Substrate.

When you run Substrate commands, you must either be in the directory where you've cloned that repository or have the fully-qualified path to that repository in the `SUBSTRATE_ROOT` environment variable.

## Get AWS credentials from your Credential Factory

No doubt you're itching to get into AWS and _do something_. Here's how:

```shell-session
eval $(substrate credentials)
```

You'll use this command every working day to get AWS credentials which last only 12 hours via your company's identity provider. It will open your web browser, where you'll authenticate, and then you'll be instructed to return to your terminal.

That web page is not just for getting AWS credentials in your terminal, though. The Accounts page on that, your company's Substrate-managed Intranet, will get you into the AWS Console in any of your AWS accounts and can be extended to wrap your own custom internal tools in the protection of your identity provider, too.

Once you have your AWS credentials, you can always run `substrate whoami` to orient yourself in your AWS organization.

## Jump into the daily Substrate workflow

Now that you've installed Substrate and Terraform, cloned your Substrate repository, and proven that you can use your identity provider to get AWS credentials, you're ready to jump into the [daily workflow](daily-workflow.md) that Substrate encourages.
