# Integrating your identity provider to control access to AWS

Substrate uses an OAuth OIDC identity provider to broker all your human access to AWS. Substrate supports [Azure Active Directory](https://azure.microsoft.com/en-us/products/active-directory/), [Google Workspace](https://workspace.google.com/) and [Okta](https://www.okta.com/). You almost certainly have one already. It pays dividends to standardize on an identity provider early and configure every bit of SaaS your company uses to rely on it.

Feel free to try them all. Whichever you choose, take comfort in knowing your decision isn't permanent. [Changing identity providers](../../runbooks/changing-identity-providers.md) documents how to move from one to another.

In the next section, whichever identity provider you choose, you'll be continuing your initial run of `substrate setup` (or running it again, which is always safe to do). The documentation will help you respond to its prompts.

One of those prompts concerns purchasing or transferring a DNS domain name or delegating a DNS zone from elsewhere into this new account. If you're at a loss for inspiration, consider using your company's name with the `.company` or `.tools` TLD. Avoid overloading any domain you use for public-facing web services, especially those that set cookies, to reduce the impact of e.g. CSRF or XSS vulnerabilities. **Be sure to respond to the emails sent by Route 53 (which Google often flags as suspicious) to avoid your domain being suspended in a few days.** This is the domain where Substrate will arrange to serve your Credential Factory for minting temporary AWS credentials, your Instance Factory for launching personal EC2 instances, your account listing which also gets you into the AWS Console, and any tools you choose to add to your Intranet later.

Jump straight to the documentation for your identity provider of choice:

* [Azure Active Directory](azure-ad.md)
* [Google Workspace](google.md)
* [Okta](okta.md)
