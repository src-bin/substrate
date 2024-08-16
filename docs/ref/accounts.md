# Accounts in a Substrate-managed AWS organization

One of the main things Substrate wants you to do is to use multiple AWS accounts. Why?

* Security: The AWS IAM team recommends using AWS accounts as the primary security boundary between unrelated systems. Within a single account it can be difficult to fully model permissions and convince oneself that two systems really are separate. Between multiple accounts, though, it's far easier to move quickly by granting broad permissions within one account with confidence that those permissions don't bleed over into the other accounts.
* Reliability: Multiple accounts enable you to implement changes domain by domain, environment by environment, and quality by quality no matter what sort of AWS resource is involved. Incremental change is fundamental to delivering reliable systems.
* Compliance: Multiple accounts make several SOC 2 control criteria very straightforward to meet.
* Cost management: Most organizations struggle to tag AWS resources effectively enough to get a clear picture of where they're spending money. Every single AWS resource, though, is unavoidably attached to some account number. By mapping AWS accounts to cost centers, one can get a baseline picture of costs that's often enough all by itself.

Substrate wholeheartedly endorses the use of multiple AWS accounts.

All your AWS accounts are listed in `substrate.accounts.txt` and on [https://example.com/accounts](https://example.com/accounts) (substituting your Intranet DNS domain name).

There are four accounts that Substrate colloquially refers to as the “special” accounts. They are:

* The **management** account, which contains no resources and is only used to create and control other accounts within the organization.
* The **audit** account, which hosts CloudTrail logs for the entire organization.
* The **deploy** account, which hosts deployable artifacts for exchange between all domains, environments, and qualities in order to allow them to remain otherwise completely separate.
* The **network** account, which hosts VPCs and subnets that are shared with other accounts to simplify network topology and reduce network transfer costs.

There are additionally **admin** accounts (of which there can be more than one), which host Intranet services like the Credential and Instance Factories, all protected by your identity provider.

Finally, there are service accounts where you host your software (be it software you've written yourself or your use of an AWS-managed service). Each of these accounts is tagged with a [domain, environment, and quality](domains-environments-qualities.md).

This constellation of AWS accounts works together to increase the reliability and security of your product and reduce the blast radius of changes to any part of it.
