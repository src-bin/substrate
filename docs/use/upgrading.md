# Upgrading Substrate

In general, upgrading Substrate is a matter of running `substrate upgrade`. If your `substrate` binary is not writeable, `substrate upgrade` will produce the URL of the tarball you'll need to download and untar to get the new `substrate` binary. You should put this in your PATH, replacing the old version.

After upgrading, you should re-run the bootstrapping and account creation commands. The most thorough upgrade requires that you run the following after replacing the binaries:

1. `substrate setup`
2. `substrate account update --domain <domain> --environment <environment> --quality <quality>` for each of your other accounts

As a convenience, `substrate account list --format shell` will generate all of these commands and put them in the proper order. For the most streamlined workflow, run `sh <(substrate account list --format shell --no-apply)`, review what Terraform plans to do, and then run `sh <(substrate account list --auto-approve --format shell)` to apply the changes.

See the [release notes](../releases.html) for version-specific upgrade instructions. They will endeavor to call out which of these steps, and potentially additional steps, are necessary to gain access to new features.

**Upgrade compatibility is only guaranteed from one month to the next so it's important to stay up-to-date. Behavior of upgrading several versions in one step is undefined and may not function properly.**
