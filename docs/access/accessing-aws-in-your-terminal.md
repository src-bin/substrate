# Accessing AWS in your terminal

For most tasks, you're going to need AWS credentials. Substrate strongly discourages the creation of personal IAM users with long-lived access keys; these are highly likely to be stolen in case of a laptop compromise and misuse can be difficult to detect. It's best not to have them at all. Instead, Substrate helps you get very short-lived AWS credentials in one of three ways:

1. Run `eval $(substrate credentials)` in your terminal and follow its instructions. This will work from instances in the cloud or a laptop, though the flow is smoothest on a laptop where a web browser can be opened from the command line. This is the best choice for most folks.
2. You can also visit [https://example.com/instance-factory](https://example.com/instance-factory) (substituting your Intranet DNS domain name) and follow the steps to launch an EC2 instance to use for your administrative work. This makes the most sense for folks who use a terminal-based text editor and like to work “in the cloud.”
3. If for some reason `eval $(substrate credentials)` doesn't work for you, visit [https://example.com/credential-factory](https://example.com/credential-factory) (substituting your Intranet DNS domain name), then paste the `export` command into your terminal. This choice will work in the widest variety of places but is the most cumbersome.

In all three cases, the temporary credentials are going to assume the role you're assigned in your identity provider (Administrator, for example) in your Substrate account. From here you'll be able to use `substrate assume-role` to move into other accounts as needed. They will be valid for 12 hours, which gives you a full day's work without reauthenticating while still being decidedly temporary.

You may also be interested in [accessing the AWS Console](accessing-the-aws-console.md).
