# Telemetry in Substrate

Substrate posts telemetry to Source & Binary when you use the command-line tools and the Intranet. The data are used to better understand how Substrate is being used and how it can be improved. The data were specifically chosen to avoid disclosing sensitive information about your organization and personally identifying information about you and your employees. Here is everything Substrate posts and how it's used:

* `Command` and `Subcommand`: The name of the Substrate tool being run, like `substrate assume-role` or `substrate-intranet` and `/accounts` (but never any custom resources added to the Intranet). These reveal how frequently each tool is used, which gives clues about people's workflows and how proactive versus reactive Substrate use is.
* `Version`: The version of Substrate being used. This reveals whether and how quickly upgrades are undertaken.
* `InitialAccountNumber` and `FinalAccountNumber`: The AWS account number(s) accessed by this command. In many cases, the two values are the same; they differ in commands like `substrate assume-role` and `substrate account adopt|create|update` which explicitly change from one account to another. [AWS account numbers are not sensitive](https://www.lastweekinaws.com/blog/are-aws-account-ids-sensitive-information/) and their variety reveals how effectively people are using AWS accounts to promote security and reliability. Additionally, posting AWS account numbers keeps the names of your domains, environments, and qualities secret since those may actually be sensitive.
* `EmailDomainName`: The domain portion of the AWS accounts' email addresses (guaranteed to be the same for all Substrate-managed AWS accounts in an organization). This associates otherwise inscrutable AWS account numbers with a customer. Posting only the domain avoids disclosing domains, environments, and qualities from the local portion of the email address since they may be sensitive.
* `EmailSHA256`: The SHA256 sum of the authenticated user's email address. This is used to approximate the total number of Substrate users without disclosing anyone's identity.
* `Prefix`: The contents of the `substrate.prefix` file that uniquely identifies your organization (but not any accounts, roles, or infrastructure within). This associates otherwise inscrutable AWS account numbers with a customer. Posting only this avoids disclosing domains, environments, and qualities and doesn't require valid AWS credentials.
* `InitialRoleName` and `FinalRoleName`: The AWS IAM role name(s) accessed by this command. In many cases, the two values are the same; they differ in commands like `substrate assume-role` and `substrate account adopt|create|update`. Substrate will only ever post “Administrator”, “Auditor”, or “Other” to avoid disclosing the names of non-Substrate-managed AWS IAM roles.
* `IsEC2`: Boolean true if the Substrate command is being executed in EC2 (or another AWS service built on EC2 and exposing the Instance Metadata Service) or boolean false if not. This reveals the prevelance of remote and CI/CD workflows.
* `OS`: The operating system of the computer running Substrate (“darwin” or “linux”).

The Telemetry is posted to [https://src-bin.com/telemetry/](https://src-bin.com/telemetry/), an endpoint hosted in AWS and Honeycomb.