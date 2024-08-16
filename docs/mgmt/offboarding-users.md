# Offboarding users

The nice thing about having an identity provider is that offboarding users from your Substrate-managed AWS organization doesn't have to involve a single additional step — just deactivate the users in your identity provider and go on about your day.

There are, however, a couple of things you might want to do to tidy up after someone leaves and loses access to AWS.

1. Look for and (probably) terminate EC2 instances they launched from the Instance Factory by their email address: `xargs -n1 aws ec2 describe-instances --filters "Name=key-name,Values=<email-address>" --region <"substrate.regions"`
2. Talk to their teammates to ensure all the infrastructure they were managing is owned and accounted for
3. Rotate secrets they may have had access to and which may be used from the public Internet e.g. GitHub API tokens, IAM user access keys for integrations that don't support IAM roles
