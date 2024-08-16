# AWS IAM Identity Center

[AWS IAM Identity Center](https://aws.amazon.com/iam/identity-center/) is AWS' own solution to many of the problems Substrate set out to solve. Read on if you want to transition your organization to use AWS IAM Identity Center alongside or instead of Substrate.

1. Make sure you've upgraded Substrate to version 2024.08.
1. Run `substrate setup` and open the AWS Console to enable AWS IAM Identity Center.
1. Re-run `substrate setup` to let it start managing AWS IAM Identity Center.
1. Visit the [AWS IAM Identity Center console](https://console.aws.amazon.com/singlesignon/home):
    * Click **Confirm identity source**, the **Actions** dropdown on that page, and **Change identity source** to configure AWS IAM Identity Center to use the same identity provider as Substrate.
    * Assign users to the Administrators and/or Auditors groups (and thus the Administrator and/or Auditor permission sets, which mimic Substrate's built-in Administrator and Auditor roles).
    * Create additional permission sets as needed.
1. Test `aws sso login` instead of `eval $(substrate credentials)` and profile blocks in `~/.aws/config` instead of `substrate assume-role`. There's no reason to switch, cold turkey, but if you're transitioning to AWS IAM Identity Center, this is a critical configuration for everyone's laptop / development environment.
