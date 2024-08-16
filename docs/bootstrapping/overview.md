# Overview

Substrate helps you manage secure, reliable, and compliant cloud infrastructure in AWS. This part of the manual exists to guide you through installing Substrate and bootstrapping your Substrate-managed AWS organization.

## A warning about existing AWS organizations

Substrate expects to be able to create and configure AWS Organizations itself. Bootstrapping Substrate in an existing AWS organization is possible but requires more finesse than simply following this guide. If you have an existing AWS organization and want to adopt Substrate, please email [hello@src-bin.com](mailto:hello@src-bin.com) so we can work with you through your adoption.

## What to expect

Following this getting started guide start to finish usually takes an hour or two, depending on how many environments and regions you define right out of the gate. You're free to work in fits and starts with the comfort of knowing that every Substrate command can be killed and restarted without losing your place or causing any harm.

You're going to need administrative privileges in your identity provider (“super admin” in Google or at least the ability to configure new apps in Okta) in the sixth step. If you don't already have that it might be wise to seek that out now before it becomes a blocker.

You're also going to be prompted for a DNS domain name to be used to serve Substrate-managed tools for minting temporary AWS credentials, launching personal EC2 instances, and accessing the AWS Console. Talk with your team now so that you know what you're going to purchase when the time comes. Here is an excerpt from a few pages ahead in the manual to help you make your choice:

> One of those prompts concerns purchasing or transferring a DNS domain name or delegating a DNS zone from elsewhere into this new account. If you're at a loss for inspiration, consider using your company's name with the `.company` or `.tools` TLD. Avoid overloading any domain you use for public-facing web services, especially those that set cookies, to reduce the impact of e.g. CSRF or XSS vulnerabilities.

Substrate's output will tell you what it's doing, what to commit to version control, and what to do next. If you're ever in doubt, get in touch in Slack or at [hello@src-bin.com](mailto:hello@src-bin.com).

After you've completed this getting started guide you'll have a fully configured AWS organization integrated with your identity provider, access to AWS via your terminal and the AWS Console protected by your identity provider, a fully configured Terraform installation, and a head start delivering secure, reliable, and compliant cloud infrastructure in AWS.
