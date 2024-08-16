# Enumerating all your AWS IAM roles

Substrate can inspect your AWS organization and all your AWS accounts to provide a higher-level view of all your AWS IAM roles than simply iterating through AWS accounts and listing all the IAM roles that exist in each one. Substrate understands how IAM roles in different accounts are related to one another.

`substrate role list` is analogous to `substrate account list`. It prints a textual representation of all the roles you've created with `substrate role create`, the accounts in which they exist, the principals who may assume the roles, and the policies that are attached.

`substrate role list --format json` provides the same data in a format that you can process programmatically.

`substrate role list --format shell` provides the same data as an executable shell program, allowing you to implement something of a continuous integration workflow with IAM roles. This is especially handy if you're adding new AWS accounts because, for example, it will create any roles created with `--domain <example>` in new (and existing) AWS accounts that were created with `--domain <example>`.
